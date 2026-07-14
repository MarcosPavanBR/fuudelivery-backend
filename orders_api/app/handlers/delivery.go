package handlers

import (
	"fmt"
	"github.com/carloshomar/fuudelivery/orders-api/app/models"
	"github.com/gofiber/fiber/v2"
)

func CalculateDeliveryValue(c *fiber.Ctx) error {

	var request struct {
		Distance        float32 `json:"distance"`
		EstablishmentID int64   `json:"establishmentId"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to parse request body"})

	}
	if request.EstablishmentID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "establishmentId is required"})
	}
	var delivery models.Delivery
	if err := models.DB.Where("establishment_id = ?", request.EstablishmentID).First(&delivery).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch delivery settings"})

	}
	// Calcular o valor da entrega
	deliveryValue := (request.Distance * delivery.PerKm) + delivery.FixedTaxa
	return c.JSON(fiber.Map{"deliveryValue": deliveryValue})
}

func InsertDelivery(c *fiber.Ctx) error {

	var request struct {
		EstablishmentID uint    `json:"establishmentId"`
		FixedTaxa       float32 `json:"fixedTaxa"`
		PerKm           float32 `json:"perKm"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to parse request body"})

	}
	newDelivery := models.Delivery{
		EstablishmentID: request.EstablishmentID,
		FixedTaxa:       request.FixedTaxa,
		PerKm:           request.PerKm,
	}
	if err := models.CreateOrUpdateDelivery(&newDelivery); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to insert or update delivery data"})

	}
	return c.JSON(fiber.Map{"delivery": newDelivery})
}

func GetDeliveryByEstablishmentID(c *fiber.Ctx) error {
	// Extrair o establishmentId dos parÃ¢metros da URL
	establishmentID := c.Params("establishmentId")
	// Converter o establishmentId para o tipo correto (int64)
	var id int64
	if _, err := fmt.Sscanf(establishmentID, "%d", &id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid establishmentId format"})

	}
	// Buscar as informaÃ§Ãµes de entrega no banco de dados
	var delivery models.Delivery
	if err := models.DB.Where("establishment_id = ?", id).First(&delivery).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Delivery settings not found for the establishment"})

	}
	return c.JSON(delivery)
}
