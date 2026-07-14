package main

import (
	"context"
	"github.com/carloshomar/fuudelivery/orders-api/app/handlers"
	"github.com/carloshomar/fuudelivery/orders-api/app/models"
	"github.com/carloshomar/fuudelivery/orders-api/app/routes"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var clients = make(map[int64]*websocket.Conn)
var clientsMu sync.Mutex

func sendMessageToClient(clientID int64, message []byte) error {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	if client, ok := clients[clientID]; ok {
		return client.WriteMessage(websocket.TextMessage, message)
	}
	return nil
}

func startQueueListener(stop <-chan struct{}) {
	dsn := os.Getenv("RABBIT_CONNECTION")
	queueName := os.Getenv("RABBIT_ORDER_QUEUE")
	var conn *amqp.Connection
	var err error
	for {
		conn, err = amqp.Dial(dsn)
		if err == nil {
			break
		}
		log.Printf("Erro ao conectar ao servidor de mensagens: %s. Tentando novamente em 5 segundos...", err)
		select {
		case <-stop:
			return
		case <-time.After(5 * time.Second):
		}
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Erro ao abrir canal: %s", err)
	}
	defer ch.Close()
	queue, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Erro ao declarar a fila: %s", err)
	}
	msgs, err := ch.Consume(
		queue.Name,
		"orders_api",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Erro ao registrar o consumidor: %s", err)
	}
	for {
		select {
		case <-stop:
			log.Println("Encerrando queue listener...")
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			bodyStr := string(msg.Body)
			handlers.ReceiveMessage(bodyStr, sendMessageToClient)
		}
	}
}

func startHTTPServer() *fiber.App {
	models.ConnectPostgresDatabase()
	app := fiber.New()
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "orders-api"})
	})
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/:id", websocket.New(func(c *websocket.Conn) {
		clientIDStr := c.Params("id")
		clientID, _ := strconv.ParseInt(clientIDStr, 10, 64)
		clientsMu.Lock()
		clients[clientID] = c
		clientsMu.Unlock()
		defer func() {
			clientsMu.Lock()
			delete(clients, clientID)
			clientsMu.Unlock()
		}()
		var (
			mt  int
			msg []byte
			err error
		)
		for {
			if mt, msg, err = c.ReadMessage(); err != nil {
				log.Printf("WebSocket read error: %s", err)
				break
			}
			if err = c.WriteMessage(mt, msg); err != nil {
				log.Printf("WebSocket write error: %s", err)
				break
			}
		}
	}))
	routes.SetupRoutes(app, sendMessageToClient)
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3000"
		}
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Erro ao iniciar servidor HTTP: %s", err)
		}
	}()
	return app
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}
	app := startHTTPServer()
	stop := make(chan struct{})
	go startQueueListener(stop)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Encerrando servidor...")
	close(stop)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("Erro ao encerrar servidor: %s", err)
	}
	log.Println("Servidor encerrado com sucesso")
}
