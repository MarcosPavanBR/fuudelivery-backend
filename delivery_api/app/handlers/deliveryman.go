package handlers

import (
	"encoding/json"
	"github.com/carloshomar/fuudelivery/delivery-api/app/dto"
	"github.com/carloshomar/fuudelivery/delivery-api/app/models"
	"github.com/gofiber/fiber/v2"
	"log"
	"strconv"
)

func GetOrdersByDeliverymanID(c *fiber.Ctx) error {
	deliverymanIDStr := c.Params("id")
	deliverymanID, err := strconv.ParseInt(deliverymanIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID de deliveryman invÃ¡lido"})

	}
	var records []models.SolicitationRecord
	if err := models.DB.Where(
		"deliveryman_id = ? AND status != ? AND deliveryman_status != ?",
		deliverymanID, "FINISHED", "FINISHED",
	).Find(&records).Error; err != nil {
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

func GetOrderByID(orderID string) (*dto.OrderDTO, error) {

	var record models.SolicitationRecord
	if err := models.DB.First(&record, "order_id = ?", orderID).Error; err != nil {
		log.Printf("Erro ao consultar o pedido: %s", err)
		return nil, err

	}
	var order dto.OrderDTO
	if err := json.Unmarshal([]byte(record.Payload), &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func UpdateOrderStatusByDeliverymanID(c *fiber.Ctx, sendMessageToClient func(clientID int64, message []byte) error) error {

	var request struct {
		OrderID     string `json:"order_id"`
		Deliveryman struct {
			Id     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"deliveryman"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Erro ao fazer parsing do corpo da requisiÃ§Ã£o"})

	}
	var record models.SolicitationRecord
	if err := models.DB.Where(
		"order_id = ? AND deliveryman_id = ?", request.OrderID, request.Deliveryman.Id,
	).First(&record).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Pedido nÃ£o encontrado"})

	}
	record.DeliverymanStatus = request.Deliveryman.Status

	var orderDTO dto.OrderDTO
	json.Unmarshal([]byte(record.Payload), &orderDTO)
	orderDTO.DeliveryMan.Status = request.Deliveryman.Status
	payloadBytes, _ := json.Marshal(&orderDTO)
	record.Payload = string(payloadBytes)
	models.DB.Save(&record)
	order, _ := GetOrderByID(request.OrderID)
	orderBytes, _ := json.Marshal(order)
	PublishMessage(orderBytes)
	return c.JSON(fiber.Map{"message": "Status do pedido atualizado com sucesso"})
}
