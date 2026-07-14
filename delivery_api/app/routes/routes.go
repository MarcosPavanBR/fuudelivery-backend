package routes

import (
	"github.com/carloshomar/fuudelivery/delivery-api/app/handlers"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(router fiber.Router, sendMessageToClient func(clientID int64, message []byte) error) {
	router.Get("/solicitation-orders", handlers.GetApprovedSolicitations)
}

func SetupProtectedRoutes(router fiber.Router, sendMessageToClient func(clientID int64, message []byte) error) {
	router.Put("/solicitation-orders/hand-shake", handlers.HandShakeDeliveryman)
	router.Get("/deliveryman/has-active/:id", handlers.GetOrdersByDeliverymanID)
	router.Post("/deliveryman/status", func(c *fiber.Ctx) error { return handlers.UpdateOrderStatusByDeliverymanID(c, sendMessageToClient) })
	router.Get("/deliveryman/extrato/:id", handlers.GetExtrato)
}
