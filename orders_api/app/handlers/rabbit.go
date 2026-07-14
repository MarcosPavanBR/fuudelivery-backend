package handlers

import (
	"encoding/json"
	"github.com/carloshomar/fuudelivery/orders-api/app/dto"
	"github.com/carloshomar/fuudelivery/orders-api/app/models"
	"github.com/streadway/amqp"
	"log"
	"os"
)

func ReceiveMessage(msg string, sendMessageToClient func(clientID int64, message []byte) error) {

	var orderMsg dto.RequestPayload
	err := json.Unmarshal([]byte(msg), &orderMsg)
	if err != nil {
		log.Printf("Erro ao decodificar a mensagem JSON: %s", err)
		return

	}
	var record models.OrderRecord
	err = models.DB.First(&record, "id = ?", orderMsg.OrderId).Error
	if err != nil {
		log.Printf("Erro ao buscar o pedido no Postgres: %s", err)
		return

	}
	var order dto.RequestPayload
	err = json.Unmarshal([]byte(record.Payload), &order)
	if err != nil {
		log.Printf("Erro ao decodificar o payload existente: %s", err)
		return

	}
	order.DeliveryMan = orderMsg.DeliveryMan
	if orderMsg.DeliveryMan.Status == "FINISHED" {
		order.Status = "FINISHED"
		record.Status = "FINISHED"

	}
	orderBytes, err := json.Marshal(&order)
	if err != nil {
		log.Printf("Erro ao codificar o payload atualizado: %s", err)
		return

	}
	record.Payload = string(orderBytes)
	err = models.DB.Save(&record).Error
	if err != nil {
		log.Printf("Erro ao atualizar o pedido no Postgres: %s", err)
		return

	}
	if orderMsg.DeliveryMan.Status == "FINISHED" {
		PublishMessage(orderBytes)
	}
	sendMessageToClient(orderMsg.EstablishmentId, orderBytes)
	log.Println("Documento atualizado com sucesso no Postgres.")
}

func PublishMessage(body []byte) error {
	// Conectar ao servidor RabbitMQ
	dsn := os.Getenv("RABBIT_CONNECTION")
	if dsn == "" {
		panic("RABBIT_CONNECTION nÃ£o configurado")
	}
	queueName := os.Getenv("RABBIT_DELIVERY_QUEUE")
	if queueName == "" {
		panic("RABBIT_DELIVERY_QUEUE nÃ£o configurado")
	}
	conn, err := amqp.Dial(dsn)
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	_, err = ch.QueueDeclare(
		queueName,
		true,  // Durable
		false, // Delete when unused
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		return err
	}
	// Publicar a mensagem na fila
	err = ch.Publish(
		"",        // Exchange
		queueName, // Routing key
		false,     // Mandatory
		false,     // Immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		})
	if err != nil {
		return err
	}
	return nil
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
