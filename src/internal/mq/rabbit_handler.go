package mq

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/cfg"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type RabbitHandler struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	exchange string
	logger   *zap.SugaredLogger
}

func NewRabbit(cfg cfg.RabbitCfg, logger *zap.SugaredLogger) (*RabbitHandler, error) {
	url := normalizeRabbitURL(cfg.Endpoint, cfg.UserName, cfg.Password)
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial failed: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq channel open failed: %w", err)
	}

	exchange := cfg.Exchange
	if strings.TrimSpace(exchange) == "" {
		exchange = "amq.topic"
	}
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq exchange declare failed: %w", err)
	}

	logger.Infof("Connected to RabbitMQ at %s", cfg.Endpoint)
	return &RabbitHandler{conn: conn, ch: ch, exchange: exchange, logger: logger}, nil
}

func (r *RabbitHandler) Publish(topic string, payload []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return r.ch.PublishWithContext(ctx, r.exchange, topic, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        payload,
		Timestamp:   time.Now(),
	})
}

func (r *RabbitHandler) Subscribe(topics []string, handler func(topic string, payload []byte)) error {
	q, err := r.ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq queue declare failed: %w", err)
	}

	for _, topic := range topics {
		if err := r.ch.QueueBind(q.Name, topic, r.exchange, false, nil); err != nil {
			return fmt.Errorf("rabbitmq queue bind failed for %s: %w", topic, err)
		}
	}

	msgs, err := r.ch.Consume(q.Name, "", true, true, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq consume failed: %w", err)
	}

	go func() {
		for m := range msgs {
			handler(m.RoutingKey, m.Body)
		}
	}()

	return nil
}

func (r *RabbitHandler) Close() {
	if r.ch != nil {
		_ = r.ch.Close()
	}
	if r.conn != nil {
		_ = r.conn.Close()
	}
}

func normalizeRabbitURL(endpoint, user, pass string) string {
	e := strings.TrimSpace(endpoint)
	if strings.HasPrefix(e, "amqp://") || strings.HasPrefix(e, "amqps://") {
		return e
	}
	if user != "" {
		if pass != "" {
			return fmt.Sprintf("amqp://%s:%s@%s/", user, pass, e)
		}
		return fmt.Sprintf("amqp://%s@%s/", user, e)
	}
	return "amqp://" + e + "/"
}
