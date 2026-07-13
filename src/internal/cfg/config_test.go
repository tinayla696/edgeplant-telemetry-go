package cfg

import (
	"path/filepath"
	"testing"
)

func TestLoadConfig_BackwardCompatibleKeys(t *testing.T) {
	cfgPath := filepath.Join("..", "..", "..", "config", "config.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	can0, ok := cfg.SocketCAN["can0"]
	if !ok {
		t.Fatalf("socketcan.can0 not found")
	}
	if can0.DbcPath != "config/can0.dbc" {
		t.Fatalf("expected can0 dbc path to be normalized, got %q", can0.DbcPath)
	}

	if cfg.Gpsd.Endpoint != "127.0.0.1:2947" {
		t.Fatalf("expected gps endpoint fallback from gpsdaemon, got %q", cfg.Gpsd.Endpoint)
	}

	if cfg.Telemetry.PublishMsgs.IntervalMs != 100 {
		t.Fatalf("expected interval fallback from interva_ms=100, got %d", cfg.Telemetry.PublishMsgs.IntervalMs)
	}

	if len(cfg.Telemetry.SubscribeMsgs.TopicPrefixes) != 1 {
		t.Fatalf("expected one topic prefix, got %d", len(cfg.Telemetry.SubscribeMsgs.TopicPrefixes))
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	t.Setenv("TELEMETRY_BROKER_TYPE", "rabbitmq")
	t.Setenv("TELEMETRY_MQTT_ENDPOINT", "mq.example.com:1883")
	t.Setenv("TELEMETRY_RABBITMQ_URL", "amqps://rabbit.example.com:5671/")

	cfgPath := filepath.Join("..", "..", "..", "config", "config.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Broker.Type != "rabbitmq" {
		t.Fatalf("expected broker type override, got %s", cfg.Broker.Type)
	}
	if cfg.Mosquitto.Endpoint != "mq.example.com:1883" {
		t.Fatalf("expected mqtt endpoint override, got %s", cfg.Mosquitto.Endpoint)
	}
	if cfg.RabbitMQ.Endpoint != "amqps://rabbit.example.com:5671/" {
		t.Fatalf("expected rabbitmq endpoint override, got %s", cfg.RabbitMQ.Endpoint)
	}
}
