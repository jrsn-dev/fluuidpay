package messaging

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/jrsn-dev/fluuidpay/ecommerce-service/internal/usecase"
)

type PaymentProcessedEvent struct {
	PaymentID     string    `json:"payment_id"`
	OrderID       string    `json:"order_id"`
	Status        string    `json:"status"` // APPROVED, REJECTED
	ProcessedAt   time.Time `json:"processed_at"`
}

type Consumer struct {
	conn *amqp.Connection
	repo usecase.EcommerceRepository
}

func NewConsumer(amqpURL string, repo usecase.EcommerceRepository) (*Consumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, err
	}
	return &Consumer{
		conn: conn,
		repo: repo,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// Ensure exchange and queue exist
	err = ch.ExchangeDeclare(
		"payment.events", // name
		"topic",          // type
		true,             // durable
		false,            // auto-deleted
		false,            // internal
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		return err
	}

	q, err := ch.QueueDeclare(
		"ecommerce.payment.processed", // name
		true,                          // durable
		false,                         // delete when unused
		false,                         // exclusive
		false,                         // no-wait
		nil,                           // arguments
	)
	if err != nil {
		return err
	}

	err = ch.QueueBind(
		q.Name,
		"payment.processed",
		"payment.events",
		false,
		nil,
	)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		q.Name,
		"ecommerce-consumer",
		false, // auto-ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Println("Ecommerce Consumer started listening for payment.processed events")

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-msgs:
			var event PaymentProcessedEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("Failed to unmarshal event: %v", err)
				msg.Nack(false, false) // send to DLQ in real world
				continue
			}

			// Map Payment Status to Order Status
			orderStatus := "PENDING"
			if event.Status == "APPROVED" {
				orderStatus = "PAID"
			} else if event.Status == "REJECTED" {
				orderStatus = "REJECTED"
			}

			log.Printf("Received payment update for order %s. New status: %s", event.OrderID, orderStatus)
			err = c.repo.UpdateOrderStatus(context.Background(), event.OrderID, orderStatus)
			if err != nil {
				log.Printf("Failed to update order %s: %v", event.OrderID, err)
				msg.Nack(false, true) // requeue
				continue
			}

			msg.Ack(false)
		}
	}
}

func (c *Consumer) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
