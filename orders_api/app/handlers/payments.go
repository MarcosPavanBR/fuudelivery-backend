package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/carloshomar/fuudelivery/orders-api/app/dto"
	"github.com/carloshomar/fuudelivery/orders-api/app/models"
	"github.com/gofiber/fiber/v2"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

var SendWSMessage func(clientID int64, message []byte) error

type AbacatePayItem struct {
	ExternalID string `json:"externalId"`
	Name       string `json:"name"`
	Quantity   int    `json:"quantity"`
	Price      int64  `json:"price"`
}

type AbacatePayCreateBillingRequest struct {
	Frequency     string            `json:"frequency"`
	Methods       []string          `json:"methods"`
	Items         []AbacatePayItem  `json:"items"`
	Metadata      map[string]string `json:"metadata"`
	ReturnURL     string            `json:"returnUrl"`
	CompletionURL string            `json:"completionUrl"`
}

type AbacatePayCreateBillingResponse struct {
	Success bool `json:"success"`
	Data    struct {
		ID     string `json:"id"`
		URL    string `json:"url"`
		Status string `json:"status"`
	} `json:"data"`
	Error string `json:"error"`
}

func mapPaymentMethods(paymentType string) []string {
	switch paymentType {
	case "pix":
		return []string{"PIX"}
	case "credit", "debit":
		return []string{"CARD"}
	default:
		return []string{"PIX"}
	}
}

func calculateOrderTotal(payload dto.RequestPayload) float64 {
	var total float64
	for _, cartItem := range payload.Cart {
		var product models.Product
		if err := models.DB.First(&product, cartItem.Item.ID).Error; err != nil {
			continue
		}
		itemTotal := product.Price
		for _, selectedID := range cartItem.Additionals {
			var add models.Additional
			if err := models.DB.First(&add, selectedID).Error; err != nil {
				continue
			}
			itemTotal += add.Price
		}
		total += itemTotal * float64(cartItem.Quantity)
	}
	var delivery models.Delivery
	if err := models.DB.Where("establishment_id = ?", payload.EstablishmentId).First(&delivery).Error; err == nil {
		total += float64(delivery.FixedTaxa) + float64(delivery.PerKm)*payload.Distance
	}
	return total
}

func CreateAbacatePayBilling(orderID string, payload dto.RequestPayload) (string, error) {
	apiKey := os.Getenv("ABACATEPAY_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ABACATEPAY_API_KEY nÃ£o configurada")
	}
	totalCents := int64(math.Round(calculateOrderTotal(payload) * 100))
	if totalCents <= 0 {
		return "", fmt.Errorf("valor do pedido invÃ¡lido: %d", totalCents)
	}
	methods := mapPaymentMethods(payload.PaymentMethod.Type)
	billingRequest := AbacatePayCreateBillingRequest{
		Frequency: "ONE_TIME",
		Methods:   methods,
		Items: []AbacatePayItem{
			{
				ExternalID: orderID,
				Name:       fmt.Sprintf("Pedido #%s - %s", orderID[:8], payload.Establishment.Name),
				Quantity:   1,
				Price:      totalCents,
			},
		},
		Metadata: map[string]string{"orderId": orderID},
		ReturnURL: func() string {
			if v := os.Getenv("ABACATEPAY_RETURN_URL"); v != "" {
				return v
			}
			return ""
		}(),
		CompletionURL: func() string {
			if v := os.Getenv("ABACATEPAY_COMPLETION_URL"); v != "" {
				return v
			}
			return ""
		}(),
	}
	jsonData, err := json.Marshal(billingRequest)
	if err != nil {
		return "", fmt.Errorf("erro ao serializar billing request: %w", err)
	}
	req, err := http.NewRequest("POST", "https://api.abacatepay.com/v2/checkouts/create", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("erro ao criar requisiÃ§Ã£o AbacatePay: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro ao chamar AbacatePay API: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("AbacatePay API error (status %d): %s", resp.StatusCode, string(bodyBytes))
		return "", fmt.Errorf("AbacatePay retornou status %d", resp.StatusCode)

	}
	var billingResponse AbacatePayCreateBillingResponse
	if err := json.Unmarshal(bodyBytes, &billingResponse); err != nil {
		return "", fmt.Errorf("erro ao decodificar resposta AbacatePay: %w", err)
	}
	if !billingResponse.Success {
		return "", fmt.Errorf("AbacatePay falhou: %s", billingResponse.Error)
	}
	return billingResponse.Data.URL, nil
}

func verifySignature(payload []byte, signature string, secret string) bool {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	expectedSignature := hex.EncodeToString(h.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(expectedSignature), []byte(signature)) == 1
}

func processOrderPayment(orderID string, record *models.OrderRecord, payload *dto.RequestPayload) {
	payload.Status = "APPROVED"
	record.Status = "APPROVED"
	orderBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling order payload: %s", err)
		return

	}
	record.Payload = string(orderBytes)
	models.DB.Save(record)
	PublishMessage(orderBytes)
	if SendWSMessage != nil {
		SendWSMessage(payload.EstablishmentId, orderBytes)
	}
	log.Printf("Order %s updated to APPROVED via Webhook", orderID)
}

func HandleWebhook(c *fiber.Ctx) error {
	secret := os.Getenv("ABACATEPAY_WEBHOOK_SECRET")
	if secret == "" {
		log.Println("ABACATEPAY_WEBHOOK_SECRET not configured")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Webhook secret not configured"})

	}
	signature := c.Get("X-Webhook-Signature")
	if signature == "" {
		log.Println("Missing X-Webhook-Signature header")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing signature"})

	}
	body := c.Body()
	if !verifySignature(body, signature, secret) {
		log.Println("Webhook signature verification failed")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid signature"})

	}
	var event struct {
		ID         string `json:"id"`
		Event      string `json:"event"`
		ApiVersion int    `json:"apiVersion"`
		DevMode    bool   `json:"devMode"`
		Data       struct {
			ID       string            `json:"id"`
			Status   string            `json:"status"`
			Metadata map[string]string `json:"metadata"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("Error unmarshalling webhook: %s", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})

	}
	log.Printf("Received webhook event: %s for billing: %s", event.Event, event.Data.ID)
	orderID := event.Data.Metadata["orderId"]
	if orderID == "" {
		log.Printf("Webhook missing orderId in metadata")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Missing orderId in metadata"})

	}
	var record models.OrderRecord
	if err := models.DB.First(&record, "id = ?", orderID).Error; err != nil {
		log.Printf("Order not found for webhook: %s", orderID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Order not found"})

	}
	var payload dto.RequestPayload
	if err := json.Unmarshal([]byte(record.Payload), &payload); err != nil {
		log.Printf("Error decoding order payload: %s", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Invalid payload in DB"})

	}
	switch {
	case event.Data.Status == "PAID":
		if payload.Status != "APPROVED" && payload.Status != "IN_PRODUCTION" {
			processOrderPayment(orderID, &record, &payload)
		}
	case event.Data.Status == "REFUNDED":
		if payload.Status != "REFUNDED" {
			payload.Status = "REFUNDED"
			record.Status = "REFUNDED"
			orderBytes, _ := json.Marshal(&payload)
			record.Payload = string(orderBytes)
			models.DB.Save(&record)
			log.Printf("Order %s refunded via Webhook", orderID)

		}

	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
}
