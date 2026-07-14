package handlers

import (
	"encoding/json"
	"github.com/carloshomar/fuudelivery/delivery-api/app/dto"
	"github.com/carloshomar/fuudelivery/delivery-api/app/models"
	"github.com/gofiber/fiber/v2"
	"log"
	"math"
	"strconv"
	"time"
)

func CreateSolicitation(msg string, sendMessageToClient func(clientID int64, message []byte) error) error {

	var orderDTO dto.OrderDTO
	err := json.Unmarshal([]byte(msg), &orderDTO)
	if err != nil {
		log.Printf("Erro ao decodificar a mensagem JSON: %s", err)
		return nil
	}
	// Verificar se jÃ¡ existe
	var existing models.SolicitationRecord
	result := models.DB.First(&existing, "order_id = ?", orderDTO.OrderId)
	if result.Error == nil {
		// JÃ¡ existe, atualizar status
		var existingDTO dto.OrderDTO
		json.Unmarshal([]byte(existing.Payload), &existingDTO)
		orderDTO.DeliveryMan = existingDTO.DeliveryMan
		existing.Status = orderDTO.Status
		existing.OperationDate = time.Now()
		orderDTO.OperationDate = existing.OperationDate
		payloadBytes, _ := json.Marshal(&orderDTO)
		existing.Payload = string(payloadBytes)
		log.Printf("Atualizando pedido %s", orderDTO.OrderId)
		log.Printf("Para Status: %s", orderDTO.Status)
		models.DB.Save(&existing)
		jsonData, _ := json.Marshal(&orderDTO)
		sendMessageToClient(orderDTO.DeliveryMan.Id, jsonData)
		return nil
	}
	// Criar novo registro
	orderDTO.OperationDate = time.Now()
	payloadBytes, _ := json.Marshal(&orderDTO)
	record := models.SolicitationRecord{
		OrderID:           orderDTO.OrderId,
		DeliverymanID:     0,
		Status:            orderDTO.Status,
		DeliverymanStatus: "",
		Payload:           string(payloadBytes),
		OperationDate:     orderDTO.OperationDate,
	}
	if err := models.DB.Create(&record).Error; err != nil {
		log.Printf("Erro ao inserir solicitaÃ§Ã£o: %s", err)
	}
	return nil
}

func HandShakeDeliveryman(c *fiber.Ctx) error {

	var orderDTO dto.OrderDTO
	if err := c.BodyParser(&orderDTO); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Erro ao fazer parsing do corpo da requisiÃ§Ã£o"})

	}
	var record models.SolicitationRecord
	if err := models.DB.First(&record, "order_id = ?", orderDTO.OrderId).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Pedido nÃ£o encontrado"})

	}
	var existingOrder dto.OrderDTO
	json.Unmarshal([]byte(record.Payload), &existingOrder)
	if existingOrder.DeliveryMan != (dto.DeliveryMan{}) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "O deliveryman jÃ¡ foi atribuÃ­do a este pedido"})

	}
	orderDTO.DeliveryMan.Status = "IN_ROUTE_COLECT"
	existingOrder.DeliveryMan = orderDTO.DeliveryMan
	payloadBytes, _ := json.Marshal(&existingOrder)
	record.Payload = string(payloadBytes)
	record.DeliverymanID = orderDTO.DeliveryMan.Id
	record.DeliverymanStatus = "IN_ROUTE_COLECT"
	models.DB.Save(&record)
	order, _ := GetOrderByID(orderDTO.OrderId)
	orderBytes, _ := json.Marshal(order)
	PublishMessage(orderBytes)
	return c.JSON(fiber.Map{"message": "Pedido atualizado com sucesso"})
}

func GetApprovedSolicitations(c *fiber.Ctx) error {
	lat := c.Query("latitude")
	long := c.Query("longitude")
	limitDistance := c.Query("limitDistance")
	latitude, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		return err
	}
	longitude, err := strconv.ParseFloat(long, 64)
	if err != nil {
		return err
	}
	limitDist, err := strconv.ParseFloat(limitDistance, 64)
	if err != nil {
		return err
	}
	var records []models.SolicitationRecord
	if err := models.DB.Where(
		"status IN ? AND (deliveryman_id = 0 OR deliveryman_id IS NULL)",
		[]string{"APPROVED", "DONE"},
	).Find(&records).Error; err != nil {
		return err
	}
	var approvedSolicitations []dto.OrderDTO
	for _, r := range records {

		var orderDTO dto.OrderDTO
		if err := json.Unmarshal([]byte(r.Payload), &orderDTO); err != nil {
			continue
		}
		distance := calculateDistance(latitude, longitude, orderDTO.Establishment.Lat, orderDTO.Establishment.Long)
		if distance <= limitDist {
			approvedSolicitations = append(approvedSolicitations, orderDTO)
		}

	}
	if approvedSolicitations == nil {
		approvedSolicitations = []dto.OrderDTO{}

	}
	return c.JSON(approvedSolicitations)
}

// FunÃ§Ã£o para calcular a distÃ¢ncia entre dois pontos usando a fÃ³rmula de Haversine (https://pt.wikipedia.org/wiki/F%C3%B3rmula_de_haversine)
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {

	const earthRadius = 6371 // Raio da Terra em quilÃ´metros
	// Converte as coordenadas de graus para radianos
	lat1Rad := degreesToRadians(lat1)
	lon1Rad := degreesToRadians(lon1)
	lat2Rad := degreesToRadians(lat2)
	lon2Rad := degreesToRadians(lon2)
	// Calcula as diferenÃ§as de coordenadas
	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad
	// Calcula as distÃ¢ncia usando a fÃ³rmula de Haversine
	a := math.Pow(math.Sin(deltaLat/2), 2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Pow(math.Sin(deltaLon/2), 2)
	cc := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadius * cc
	return distance
}

// FunÃ§Ã£o para converter graus em radianos
func degreesToRadians(degrees float64) float64 { return degrees * math.Pi / 180 }
