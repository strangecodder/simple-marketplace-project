package rabbit

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// todo: добавить структуру RabbitConfig
type RabbitProducer interface {
	Publish(queueName string, body []byte) error
}

type RabbitClient struct {
	Conn   *amqp.Connection
	Ch     *amqp.Channel
	logger *slog.Logger
}

func NewRabbitClient(logger *slog.Logger) *RabbitClient {
	rabbitClient := &RabbitClient{logger: logger}
	return rabbitClient
}

func (r *RabbitClient) Connect(url string) error {
	var conn *amqp.Connection
	var err error

	maxRetries := 10
	for i := 1; i <= maxRetries; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("rabbitmq connect attempt %d/%d failed: %v", i, maxRetries, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to rabbitmq after %d attempts: %w", maxRetries, err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	r.Conn = conn
	r.Ch = ch

	return nil
}

func (r *RabbitClient) Close() {
	if r.Ch != nil {
		r.Ch.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
}

func (r *RabbitClient) DeclareQueue(name string) error {
	_, err := r.Ch.QueueDeclare(
		name,
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	return nil
}

type HandlerFunc func(ctx context.Context, body []byte) error

// Consume запускает consumer и передаёт каждое сообщение в handler
func (r *RabbitClient) Consume(ctx context.Context, queueName string, handler HandlerFunc) error {
	msgs, err := r.Ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("consumer for queue %q stopped", queueName)
				return
			case msg, ok := <-msgs:
				if !ok {
					log.Printf("channel for queue %q closed", queueName)
					return
				}
				if err := handler(ctx, msg.Body); err != nil {
					log.Printf("handler error for queue %q: %v", queueName, err)
					msg.Nack(false, true)
					continue
				}
				msg.Ack(false)
			}
		}
	}()

	return nil
}

func (r *RabbitClient) Publish(queueName string, body []byte) error {
	return r.Ch.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
}
