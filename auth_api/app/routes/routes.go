package routes

import (
	"github.com/carloshomar/fuudelivery/auth-api/app/handlers"
	"github.com/carloshomar/fuudelivery/auth-api/app/middlewares"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(router fiber.Router) {
	router.Post("/users/register", handlers.CreateUser)
	router.Post("/users/login", handlers.Login)
	router.Get("/establishments", handlers.ListEstablishments)
	router.Get("/establishments/:id", handlers.GetEstablishments)
	router.Post("/delivery-man/login", handlers.LoginDeliveryMan)
	router.Post("/delivery-man/register", handlers.CreateDeliveryMan)
}

func SetupProtectedRoutes(router fiber.Router) {
	router.Get("/users/:id", handlers.GetUser)
	router.Put("/establishments/status/handler/:id", handlers.HandlerEstablishmentStatus)
	router.Put("/establishments/:id", handlers.UpdateEstablishment)
	router.Get("/establishments/:id/users", handlers.GetUserByEstablishment)
}

func ProtectedRoute(c *fiber.Ctx) error {
	_, err := middlewares.ValidateJWT(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}
	return c.Next()
}
