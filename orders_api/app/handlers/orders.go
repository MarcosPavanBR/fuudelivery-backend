package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/carloshomar/fuudelivery/orders-api/app/dto"
	"github.com/carloshomar/fuudelivery/orders-api/app/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

func CreateOrder(c *fiber.Ctx, sendMessageToClient func(clientID int64, message []byte) error) error {

	var request dto.RequestPayload
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Erro ao fazer parsing do corpo da requisiÃ§Ã£o"})

	}
	request.Status = "AWAIT_APPROVE"
	establishment, err := GetEstablishment(request.EstablishmentId)
	if err != nil {
		log.Printf("Aviso: nao foi possivel obter detalhes do estabelecimento %d: %s", request.EstablishmentId, err)
		establishment = &dto.Establishment{
			Id:   request.EstablishmentId,
			Name: "FuuDelivery",
		}
	}
	request.Establishment = *establishment
	orderID := uuid.New().String()
	request.OrderId = orderID
	payloadBytes, err := json.Marshal(request)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Erro ao serializar o pedido"})

	}
	record := models.OrderRecord{
		ID:              orderID,
		EstablishmentID: request.EstablishmentId,
		Phone:           request.User.Phone,
		Status:          request.Status,
		Payload:         string(payloadBytes),
		LastModified:    time.Now(),
	}
	// Se o pagamento for digital (pix ou credit/debit), criar checkout na AbacatePay ANTES de salvar
	if request.PaymentMethod.
		Type == "pix" || request.PaymentMethod.
		Type == "credit" || request.PaymentMethod.
		Type == "debit" {
		paymentURL, err := CreateAbacatePayBilling(orderID, request)
		if err != nil {
			log.Printf("Erro ao criar billing no AbacatePay: %s", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Falha ao processar pagamento",
			})

		}
		request.PaymentURL = paymentURL
		payloadBytes, _ = json.Marshal(request)
		record.Payload = string(payloadBytes)

	}
	if err := models.DB.Create(&record).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Erro ao inserir a ordem no banco de dados"})

	}
	if err := sendMessageToClient(request.EstablishmentId, payloadBytes); err != nil {
		log.Printf("Erro ao enviar mensagem WebSocket: %s", err)
	}
	response := fiber.Map{
		"message": "Ordem criada com sucesso",
		"orderId": orderID,
	}
	if request.PaymentURL != "" {
		response["payment_url"] = request.PaymentURL
	}
	return c.JSON(response)
}

func GetEstablishment(establishmentID int64) (*dto.Establishment, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	urls := []string{
		fmt.Sprintf("http://localhost:%s/api/auth/establishments/%d", port, establishmentID),
	}
	urlEnv := os.Getenv("URL_GET_ESTABLISHMENT_ID")
	if urlEnv != "" {
		urls = append(urls, fmt.Sprintf(urlEnv, establishmentID))
	}
	var lastErr error
	for _, url := range urls {
		response, err := http.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			lastErr = fmt.Errorf("API retornou status nÃ£o OK: %d", response.StatusCode)
			continue
		}
		var establishmentDTO dto.Establishment
		if err := json.NewDecoder(response.Body).Decode(&establishmentDTO); err != nil {
			response.Body.Close()
			lastErr = err
			continue
		}
		response.Body.Close()
		return &establishmentDTO, nil
	}
	return nil, lastErr
}

func UpdateOrderStatus(c *fiber.Ctx, sendMessageToClient func(clientID int64, message []byte) error) error {

	var requestBody dto.UpdateOrderStatusRequest
	if err := c.BodyParser(&requestBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Erro ao fazer parsing do corpo da requisiÃ§Ã£o"})

	}
	var record models.OrderRecord
	if err := models.DB.First(&record, "id = ?", requestBody.ID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Nenhum pedido encontrado com o ID fornecido"})

	}
	var order dto.RequestPayload
	if err := json.Unmarshal([]byte(record.Payload), &order); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Erro ao decodificar payload do pedido"})

	}
	if requestBody.Status != "REQUEST_APPROVE" {
		order.OrderId = requestBody.ID
		order.Status = requestBody.Status
		orderBytes, err := json.Marshal(&order)
		if err == nil {
			PublishMessage(orderBytes)
		}

	}
	jsonData, _ := json.Marshal(requestBody)
	if err := sendMessageToClient(order.EstablishmentId, jsonData); err != nil {
		return err
	}
	record.Status = requestBody.Status
	order.Status = requestBody.Status
	payloadBytes, _ := json.Marshal(&order)
	record.Payload = string(payloadBytes)
	record.LastModified = time.Now()
	if err := models.DB.Save(&record).Error; err != nil {
		log.Printf("Erro ao salvar atualizaÃ§Ã£o do pedido: %s", err)
	}
	return c.JSON(fiber.Map{"message": "Status do pedido atualizado com sucesso"})
}

func ListOrdersByEstablishmentID(c *fiber.Ctx) error {
	establishmentID := c.Params("establishmentId")
	if establishmentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID do estabelecimento nÃ£o fornecido"})

	}
	establishmentIDInt, err := strconv.Atoi(establishmentID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID do estabelecimento invÃ¡lido"})

	}
	var records []models.OrderRecord
	if err := models.DB.Where("establishment_id = ?", establishmentIDInt).Find(&records).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Falha ao buscar pedidos"})

	}
	var orders []map[string]interface {
	}
	for _, r := range records {

		var o map[string]interface {
		}
		json.Unmarshal([]byte(r.Payload), &o)
		o["_id"] = r.ID
		o["lastModified"] = r.LastModified
		orders = append(orders, o)

	}
	if orders == nil {
		orders = []map[string]interface{}{}

	}
	return c.JSON(orders)
}

func ListOrdersByEstablishmentIDAndPhone(c *fiber.Ctx) error {
	establishmentID := c.Params("establishmentId")
	phoneNumberEncoded := c.Params("phoneNumber")
	if establishmentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID do estabelecimento nÃ£o fornecido"})

	}
	establishmentIDInt, err := strconv.Atoi(establishmentID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID do estabelecimento invÃ¡lido"})

	}
	phoneNumber, _ := url.QueryUnescape(phoneNumberEncoded)
	var records []models.OrderRecord
	if err := models.DB.Where("establishment_id = ? AND phone = ?", establishmentIDInt, phoneNumber).Find(&records).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Falha ao buscar pedidos"})

	}
	var orders []map[string]interface {
	}
	for _, r := range records {

		var o map[string]interface {
		}
		json.Unmarshal([]byte(r.Payload), &o)
		o["_id"] = r.ID
		o["lastModified"] = r.LastModified
		orders = append(orders, o)

	}
	if orders == nil {
		orders = []map[string]interface{}{}

	}
	return c.JSON(orders)
}

func ListOrdersByPhone(c *fiber.Ctx) error {
	phoneNumberEncoded := c.Params("phone")
	phoneNumber, err := url.QueryUnescape(phoneNumberEncoded)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Erro ao decodificar nÃºmero de telefone"})

	}
	var records []models.OrderRecord
	if err := models.DB.Where("phone = ?", phoneNumber).Order("last_modified DESC").Find(&records).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Falha ao buscar pedidos"})

	}
	var orders []map[string]interface {
	}
	for _, r := range records {

		var o map[string]interface {
		}
		json.Unmarshal([]byte(r.Payload), &o)
		o["_id"] = r.ID
		o["lastModified"] = r.LastModified
		orders = append(orders, o)

	}
	if orders == nil {
		orders = []map[string]interface{}{}

	}
	return c.JSON(orders)
}
