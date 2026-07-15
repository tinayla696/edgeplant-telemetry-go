package mq

import (
	"fmt"
	"strings"

	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/cfg"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

type Broker interface {
	Publish(topic string, payload []byte) error
	Subscribe(topics []string, handler func(topic string, payload []byte)) error
	Close()
}

func NewBroker(deviceID string, brokerType string, mqttCfg cfg.MqttCfg, rabbitCfg cfg.RabbitCfg, logger *zap.SugaredLogger) (Broker, error) {
	switch strings.ToLower(strings.TrimSpace(brokerType)) {
	case "", "mqtt", "mosquitto":
		h, err := New(deviceID, mqttCfg, logger)
		if err != nil {
			return nil, err
		}
		return &mqttBroker{h: h}, nil
	case "rabbitmq", "amqp":
		r, err := NewRabbit(rabbitCfg, logger)
		if err != nil {
			return nil, err
		}
		return r, nil
	default:
		return nil, fmt.Errorf("unsupported broker type: %s", brokerType)
	}
}

type mqttBroker struct {
	h *Handler
}

func (m *mqttBroker) Publish(topic string, payload []byte) error {
	m.h.Publish(topic, 0, false, payload)
	return nil
}

func (m *mqttBroker) Subscribe(topics []string, handler func(topic string, payload []byte)) error {
	topicMap := make(map[string]byte, len(topics))
	for _, topic := range topics {
		topicMap[topic] = 0
	}
	if err := m.h.Subscribe(topicMap, func(_ mqtt.Client, msg mqtt.Message) {
		handler(msg.Topic(), msg.Payload())
	}); err != nil {
		return err
	}
	return nil
}

func (m *mqttBroker) Close() {
	m.h.Close()
}
