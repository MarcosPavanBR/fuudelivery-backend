package routes

import (
	"github.com/carloshomar/fuudelivery/orders-api/app/handlers"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(router fiber.Router, sendMessageToClient func(clientID int64, message []byte) error) {
	handlers.SendWSMessage = sendMessageToClient
	router.Get("/ping", handlers.Ping)
	router.Get("/products/all/:establishmentId", handlers.GetByEstablishmentIdWithRelations)
	router.Get("/products/:establishmentId", handlers.GetByEstablishmentId)
	router.Get("/categories/:establishmentId", handlers.GetCategories)
	router.Get("/categories/product/:establishmentId", handlers.GetCategoriesWithProducts)
	router.Get("/additional/:id", handlers.ListAdditional)
	router.Get("/delivery/value/:establishmentId", handlers.GetDeliveryByEstablishmentID)
	router.Post("/delivery/calculate-delivery-value", handlers.CalculateDeliveryValue)
	router.Post("/orders", func(c *fiber.Ctx) error { return handlers.CreateOrder(c, sendMessageToClient) })
	router.Get("/orders/:establishmentId", handlers.ListOrdersByEstablishmentID)
	router.Get("/orders/list-phone/:phone", handlers.ListOrdersByPhone)
	router.Get("/orders/:establishmentId/:phoneNumber", handlers.ListOrdersByEstablishmentIDAndPhone)
	router.Post("/payments/webhook", handlers.HandleWebhook)
}

func SetupProtectedRoutes(router fiber.Router, sendMessageToClient func(clientID int64, message []byte) error) {
	handlers.SendWSMessage = sendMessageToClient
	router.Post("/products/create", handlers.CreateProduct)
	router.Delete("/products/delete/:id", handlers.DeleteProduct)
	router.Post("/products/multi-create", handlers.CreateMultProducts)
	router.Put("/products/update/:id", handlers.UpdateProduct)
	router.Post("/categories/create", handlers.CreateCategories)
	router.Post("/categories/product", handlers.CreateProductCategorie)
	router.Delete("/categories/:id", handlers.DeleteCategory)
	router.Put("/categories/:id", handlers.UpdateCategory)
	router.Post("/additional", handlers.CreateAdditional)
	router.Put("/additional/:id", handlers.UpdateAdditional)
	router.Delete("/additional/:id", handlers.DeleteAdditional)
	router.Post("/additional/product", handlers.CreateProductToAdditional)
	router.Post("/delivery", handlers.InsertDelivery)
	router.Put("/orders/status", func(c *fiber.Ctx) error { return handlers.UpdateOrderStatus(c, sendMessageToClient) })
}
