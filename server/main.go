package main

import (
	"context"
	authMiddleware "github.com/carloshomar/fuudelivery/auth-api/app/middlewares"
	authModels "github.com/carloshomar/fuudelivery/auth-api/app/models"
	authRoutes "github.com/carloshomar/fuudelivery/auth-api/app/routes"
	deliveryHandlers "github.com/carloshomar/fuudelivery/delivery-api/app/handlers"
	deliveryModels "github.com/carloshomar/fuudelivery/delivery-api/app/models"
	deliveryRoutes "github.com/carloshomar/fuudelivery/delivery-api/app/routes"
	orderHandlers "github.com/carloshomar/fuudelivery/orders-api/app/handlers"
	orderModels "github.com/carloshomar/fuudelivery/orders-api/app/models"
	orderRoutes "github.com/carloshomar/fuudelivery/orders-api/app/routes"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
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

var orderClients = make(map[int64]*websocket.Conn)
var orderClientsMu sync.Mutex

var deliveryClients = make(map[int64]*websocket.Conn)
var deliveryClientsMu sync.Mutex

func sendOrderWSMessage(clientID int64, message []byte) error {
	orderClientsMu.Lock()
	defer orderClientsMu.Unlock()
	if client, ok := orderClients[clientID]; ok {
		return client.WriteMessage(websocket.TextMessage, message)
	}
	return nil
}

func sendDeliveryWSMessage(clientID int64, message []byte) error {
	deliveryClientsMu.Lock()
	defer deliveryClientsMu.Unlock()
	if client, ok := deliveryClients[clientID]; ok {
		return client.WriteMessage(websocket.TextMessage, message)
	}
	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "https://fuudelivery.com.br,http://localhost:5173,https://fuudelivery-admin.onrender.com",
		AllowMethods: "GET,POST,PUT,DELETE",
		AllowHeaders: "Authorization,Content-Type",
	}))
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		return c.Next()
	})
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "fuudelivery-server",
		})
	})

	go func() {
		os.Setenv("DB_CONNECTION_STRING", os.Getenv("AUTH_DB_CONNECTION_STRING"))
		authModels.ConnectDatabase()
		os.Setenv("DB_CONNECTION_STRING", os.Getenv("ORDERS_DB_CONNECTION_STRING"))
		orderModels.ConnectPostgresDatabase()
		os.Setenv("DB_CONNECTION_STRING", os.Getenv("DELIVERY_DB_CONNECTION_STRING"))
		deliveryModels.ConnectDatabase()
	}()

	api := app.Group("/api")
	api.Use(limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
	}))

	authGroup := api.Group("/auth")
	authRoutes.SetupRoutes(authGroup)
	authProtected := authGroup.Group("/", authMiddleware.JWTAuth)
	authRoutes.SetupProtectedRoutes(authProtected)

	orderGroup := api.Group("/order")
	orderRoutes.SetupRoutes(orderGroup, sendOrderWSMessage)
	orderProtected := orderGroup.Group("/", authMiddleware.JWTAuth)
	orderRoutes.SetupProtectedRoutes(orderProtected, sendOrderWSMessage)

	deliveryGroup := api.Group("/delivery")
	deliveryRoutes.SetupRoutes(deliveryGroup, sendDeliveryWSMessage)
	deliveryProtected := deliveryGroup.Group("/", authMiddleware.JWTAuth)
	deliveryRoutes.SetupProtectedRoutes(deliveryProtected, sendDeliveryWSMessage)

	setupOrderWebSocket(orderGroup)
	setupDeliveryWebSocket(deliveryGroup)
	stop := make(chan struct{})
	go startOrderQueueListener(stop)
	go startDeliveryQueueListener(stop)
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Erro ao iniciar servidor: %s", err)
		}
	}()
	log.Println("Servidor iniciado na porta " + os.Getenv("PORT"))
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

func setupOrderWebSocket(group fiber.Router) {
	group.Use("/ws", func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}
		return c.Next()
	})
	group.Get("/ws/:id", websocket.New(func(c *websocket.Conn) {
		token := c.Query("token")
		if token == "" {
			c.WriteMessage(websocket.CloseMessage, []byte("missing token"))
			c.Close()
			return
		}
		clientIDStr := c.Params("id")
		clientID, _ := strconv.ParseInt(clientIDStr, 10, 64)
		orderClientsMu.Lock()
		orderClients[clientID] = c
		orderClientsMu.Unlock()
		defer func() {
			orderClientsMu.Lock()
			delete(orderClients, clientID)
			orderClientsMu.Unlock()
		}()
		for {
			mt, msg, err := c.ReadMessage()
			if err != nil {
				break
			}
			if err = c.WriteMessage(mt, msg); err != nil {
				break
			}
		}
	}))
}

func setupDeliveryWebSocket(group fiber.Router) {
	group.Use("/ws", func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}
		return c.Next()
	})
	group.Get("/ws/:id", websocket.New(func(c *websocket.Conn) {
		clientIDStr := c.Params("id")
		clientID, _ := strconv.ParseInt(clientIDStr, 10, 64)
		deliveryClientsMu.Lock()
		deliveryClients[clientID] = c
		deliveryClientsMu.Unlock()
		defer func() {
			deliveryClientsMu.Lock()
			delete(deliveryClients, clientID)
			deliveryClientsMu.Unlock()
		}()
		for {
			mt, msg, err := c.ReadMessage()
			if err != nil {
				break
			}
			if err = c.WriteMessage(mt, msg); err != nil {
				break
			}
		}
	}))
}

func startOrderQueueListener(stop <-chan struct{}) {
	dsn := os.Getenv("RABBIT_CONNECTION")
	queueName := os.Getenv("RABBIT_ORDER_QUEUE")
	var conn *amqp.Connection
	var err error
	for {
		conn, err = amqp.Dial(dsn)
		if err == nil {
			break
		}
		log.Printf("Orders queue: erro ao conectar RabbitMQ: %s. Tentando novamente...", err)
		select {
		case <-stop:
			return
		case <-time.After(5 * time.Second):
		}
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Orders queue: erro ao abrir canal: %s", err)
	}
	defer ch.Close()
	queue, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Orders queue: erro ao declarar fila: %s", err)
	}
	msgs, err := ch.Consume(queue.Name, "orders_api", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Orders queue: erro ao registrar consumidor: %s", err)
	}
	for {
		select {
		case <-stop:
			log.Println("Orders queue: encerrando...")
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			orderHandlers.ReceiveMessage(string(msg.Body), sendOrderWSMessage)
		}
	}
}

func startDeliveryQueueListener(stop <-chan struct{}) {
	dsn := os.Getenv("RABBIT_CONNECTION")
	queueName := os.Getenv("RABBIT_DELIVERY_QUEUE")
	var conn *amqp.Connection
	var err error
	for {
		conn, err = amqp.Dial(dsn)
		if err == nil {
			break
		}
		log.Printf("Delivery queue: erro ao conectar RabbitMQ: %s. Tentando novamente...", err)
		select {
		case <-stop:
			return
		case <-time.After(5 * time.Second):
		}
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Delivery queue: erro ao abrir canal: %s", err)
	}
	defer ch.Close()
	queue, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Delivery queue: erro ao declarar fila: %s", err)
	}
	msgs, err := ch.Consume(queue.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Delivery queue: erro ao registrar consumidor: %s", err)
	}
	for {
		select {
		case <-stop:
			log.Println("Delivery queue: encerrando...")
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			deliveryHandlers.CreateSolicitation(string(msg.Body), sendDeliveryWSMessage)
		}
	}
}
