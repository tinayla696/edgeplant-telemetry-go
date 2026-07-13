package mq

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/cfg"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

// MQTT handler struct
type Handler struct {
	deviceID string
	client   mqtt.Client
	logger   *zap.SugaredLogger
}

// Create a new MQTT handler
func New(deviceID string, cfg cfg.MqttCfg, logger *zap.SugaredLogger) (*Handler, error) {
	// Create MQTT client options
	opts := mqtt.NewClientOptions()
	opts.SetClientID(deviceID)
	opts.AddBroker(normalizeBrokerEndpoint(cfg.Endpoint, cfg.UseTLS))
	opts.SetUsername(cfg.UserName)
	opts.SetPassword(cfg.Password)

	// TLS configuration
	if cfg.UseTLS {
		tlsConfig, err := newTLSCfg(cfg.CertPath, cfg.ClientCert, cfg.ClientKey)
		if err != nil {
			return nil, err
		}
		opts.SetTLSConfig(tlsConfig)
	}

	// Reconnect handler
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(500 * time.Millisecond)

	// Callbacks
	opts.OnConnect = func(c mqtt.Client) {
		logger.Infof("Connected to MQTT broker at %s", cfg.Endpoint)
	}
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		logger.Warnf("Connection lost to MQTT broker: %v", err)
	}

	// Create MQTT client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("Failed to connect to MQTT broker: %v", token.Error())
	}

	return &Handler{
		deviceID: deviceID,
		client:   client,
		logger:   logger,
	}, nil
}

// TLS configuration helper function
func newTLSCfg(caCertPath, clientCertPath, clientKeyPath string) (*tls.Config, error) {
	certpool := x509.NewCertPool()
	ca, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, err
	}
	certpool.AppendCertsFromPEM(ca)

	// Load client cert
	clientKeyPair, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		RootCAs:            certpool,
		Certificates:       []tls.Certificate{clientKeyPair},
		InsecureSkipVerify: true, // This is not recommended for production, but may be necessary for self-signed certificates
	}, nil
}

// Close the MQTT client
func (h *Handler) Close() {
	h.logger.Infof("Disconnecting from MQTT broker...")
	h.client.Disconnect(250)
}

// Publish a message to a topic
func (h *Handler) Publish(topic string, qos byte, retained bool, payload []byte) {
	if !h.client.IsConnected() {
		h.logger.Warnf("MQTT client is not connected. Cannot publish to topic: %s", topic)
		return
	}

	token := h.client.Publish(topic, qos, retained, payload)
	if token.Wait() && token.Error() != nil {
		h.logger.Errorf("Failed to publish to topic %s: %v", topic, token.Error())
	}
}

// Subscribe to topics
func (h *Handler) Subscribe(topics map[string]byte, callback mqtt.MessageHandler) {
	if !h.client.IsConnected() {
		h.logger.Warnf("MQTT client is not connected. Cannot subscribe to topics.")
		return
	}

	token := h.client.SubscribeMultiple(topics, callback)
	if token.Wait() && token.Error() != nil {
		h.logger.Errorf("Failed to subscribe to topics: %v", token.Error())
	}
}

func normalizeBrokerEndpoint(endpoint string, useTLS bool) string {
	e := strings.TrimSpace(endpoint)
	if strings.HasPrefix(e, "tcp://") || strings.HasPrefix(e, "ssl://") || strings.HasPrefix(e, "tls://") || strings.HasPrefix(e, "ws://") || strings.HasPrefix(e, "wss://") {
		return e
	}
	if useTLS {
		return "ssl://" + e
	}
	return "tcp://" + e
}
