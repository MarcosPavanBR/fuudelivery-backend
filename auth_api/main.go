package main

import (
	"context"
	"github.com/carloshomar/fuudelivery/auth-api/app/models"
	"github.com/carloshomar/fuudelivery/auth-api/app/routes"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}
	models.ConnectDatabase()
	app := fiber.New()
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "auth-api"})
	})
	routes.SetupRoutes(app)
	go func() {
		if err := app.Listen(":3000"); err != nil {
			log.Fatalf("Erro ao iniciar servidor: %s", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Encerrando servidor...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("Erro ao encerrar servidor: %s", err)
	}
	log.Println("Servidor encerrado com sucesso")
}
