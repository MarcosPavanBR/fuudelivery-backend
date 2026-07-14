package handlers

import (
	"encoding/json"
	"github.com/carloshomar/fuudelivery/delivery-api/app/dto"
	"github.com/carloshomar/fuudelivery/delivery-api/app/models"
	"github.com/gofiber/fiber/v2"
	"log"
	"strconv"
)

func GetExtrato(c *fiber.Ctx) error {
	deliverymanIDStr := c.Params("id")
	deliverymanID, err := strconv.ParseInt(deliverymanIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID de deliveryman invÃ¡lido"})

	}
	var records []models.SolicitationRecord
	if err := models.DB.Where(
		"deliveryman_id = ? AND status = ? AND deliveryman_status = ?",
		deliverymanID, "FINISHED", "FINISHED",
	).Order("operation_date DESC").Find(&records).Error; err != nil {
		log.Printf("Erro ao consultar os pedidos: %s", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Erro ao consultar os pedidos"})

	}
	var orders []dto.OrderDTO
	for _, r := range records {

		var order dto.OrderDTO
		if err := json.Unmarshal([]byte(r.Payload), &order); err != nil {
			log.Printf("Erro ao decodificar o pedido: %s", err)
			continue

		}
		orders = append(orders, order)

	}
	if orders == nil {
		orders = []dto.OrderDTO{}

	}
	return c.JSON(orders)
}
