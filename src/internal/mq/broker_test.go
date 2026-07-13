package mq

import (
	"os"
	"testing"

	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/cfg"
	"go.uber.org/zap"
)

func TestNormalizeRabbitURL(t *testing.T) {
	got := normalizeRabbitURL("localhost:5672", "guest", "guest")
	want := "amqp://guest:guest@localhost:5672/"
	if got != want {
		t.Fatalf("unexpected url: got=%s want=%s", got, want)
	}
}

func TestNewBroker_UnknownType(t *testing.T) {
	_, err := NewBroker("dev1", "unknown", cfg.MqttCfg{}, cfg.RabbitCfg{}, zap.NewNop().Sugar())
	if err == nil {
		t.Fatalf("expected error for unknown broker type")
	}
}

func TestEnvOverrideSmoke(t *testing.T) {
	_ = os.Setenv("TELEMETRY_BROKER_TYPE", "rabbitmq")
	defer os.Unsetenv("TELEMETRY_BROKER_TYPE")
}
