package cfg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Application configuration struct
type AppCfg struct {
	SocketCAN map[string]SocketCanCfg `json:"socketcan" yaml:"socketcan"`
	Mosquitto MqttCfg                 `json:"mosquitto" yaml:"mosquitto"`
	RabbitMQ  RabbitCfg               `json:"rabbitmq" yaml:"rabbitmq"`
	Broker    BrokerCfg               `json:"broker" yaml:"broker"`
	Gpsd      GpsdCfg                 `json:"gpsd" yaml:"gpsd"`
	Gpsdaemon GpsdCfg                 `json:"gpsdaemon" yaml:"gpsdaemon"`
	Telemetry TelemetryCfg            `json:"telemetry_options" yaml:"telemetry_options"`
}

// SocketCAN configuration struct
type SocketCanCfg struct {
	Type          string   `json:"type" yaml:"type"`
	Bitrate       uint32   `json:"bitrate" yaml:"bitrate"`
	DbcPath       string   `json:"dbc_path" yaml:"dbc_path"`
	DbcPathLegacy string   `json:"dbcpath" yaml:"dbcpath"`
	LatchFrameIDs []uint32 `json:"latch_frame_ids" yaml:"latch_frame_ids"`
}

// MQTT configuration struct
type MqttCfg struct {
	Endpoint   string `json:"endpoint" yaml:"endpoint"`
	UserName   string `json:"username" yaml:"username"`
	Password   string `json:"password" yaml:"password"`
	UseTLS     bool   `json:"use_tls" yaml:"use_tls"`
	CertPath   string `json:"ca_cert_path" yaml:"ca_cert_path"`
	CertFile   string `json:"ca_cert_file" yaml:"ca_cert_file"`
	ClientCert string `json:"client_cert" yaml:"client_cert"`
	ClientKey  string `json:"client_key" yaml:"client_key"`
}

type RabbitCfg struct {
	Endpoint string `json:"endpoint" yaml:"endpoint"`
	UserName string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Exchange string `json:"exchange" yaml:"exchange"`
}

type BrokerCfg struct {
	Type string `json:"type" yaml:"type"`
}

// GPSD configuration struct
type GpsdCfg struct {
	Endpoint string `json:"endpoint" yaml:"endpoint"`
}

type TelemetryCfg struct {
	PublishMsgs   PublishCfg   `json:"publish_msgs" yaml:"publish_msgs"`
	SubscribeMsgs SubscribeCfg `json:"subscribe_msgs" yaml:"subscribe_msgs"`
}

type PublishCfg struct {
	TopicPrefix    string `json:"topic_prefix" yaml:"topic_prefix"`
	IntervalMs     int    `json:"interval_ms" yaml:"interval_ms"`
	IntervalMsTypo int    `json:"interva_ms" yaml:"interva_ms"`
}

type SubscribeCfg struct {
	TopicPrefixes     []string `json:"topic_prefixes" yaml:"topic_prefixes"`
	TopicPrefixesTypo []string `json:"topic_prefixs" yaml:"topic_prefixs"`
}

// LoadConfig loads the configuration from a YAML file
func LoadConfig(path string) (*AppCfg, error) {
	var cfg AppCfg

	// Read the Config File
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 拡張子のみを取得
	ext := strings.ToLower(filepath.Ext(path))

	// Switch file extension to determine the format
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(f, &cfg); err != nil {
			return nil, err
		}
	case ".json":
		if err := json.Unmarshal(f, &cfg); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}

	cfg.normalize()

	return &cfg, nil
}

func (c *AppCfg) normalize() {
	for key, canCfg := range c.SocketCAN {
		if canCfg.DbcPath == "" {
			canCfg.DbcPath = canCfg.DbcPathLegacy
		}
		c.SocketCAN[key] = canCfg
	}

	if c.Mosquitto.CertPath == "" {
		c.Mosquitto.CertPath = c.Mosquitto.CertFile
	}

	if c.Gpsd.Endpoint == "" {
		c.Gpsd.Endpoint = c.Gpsdaemon.Endpoint
	}

	if c.Telemetry.PublishMsgs.IntervalMs == 0 {
		c.Telemetry.PublishMsgs.IntervalMs = c.Telemetry.PublishMsgs.IntervalMsTypo
	}
	if c.Telemetry.PublishMsgs.IntervalMs <= 0 {
		c.Telemetry.PublishMsgs.IntervalMs = 100
	}

	if len(c.Telemetry.SubscribeMsgs.TopicPrefixes) == 0 {
		c.Telemetry.SubscribeMsgs.TopicPrefixes = c.Telemetry.SubscribeMsgs.TopicPrefixesTypo
	}

	if c.Telemetry.PublishMsgs.TopicPrefix == "" {
		c.Telemetry.PublishMsgs.TopicPrefix = "state"
	}
	if len(c.Telemetry.SubscribeMsgs.TopicPrefixes) == 0 {
		c.Telemetry.SubscribeMsgs.TopicPrefixes = []string{"ctrl"}
	}

	if c.Broker.Type == "" {
		c.Broker.Type = "mqtt"
	}
	if c.RabbitMQ.Exchange == "" {
		c.RabbitMQ.Exchange = "amq.topic"
	}

	c.applyEnvOverrides()
}

func (c *AppCfg) applyEnvOverrides() {
	if v := strings.TrimSpace(os.Getenv("TELEMETRY_BROKER_TYPE")); v != "" {
		c.Broker.Type = v
	}

	if v := strings.TrimSpace(os.Getenv("TELEMETRY_MQTT_ENDPOINT")); v != "" {
		c.Mosquitto.Endpoint = v
	}
	if v := os.Getenv("TELEMETRY_MQTT_USERNAME"); v != "" {
		c.Mosquitto.UserName = v
	}
	if v := os.Getenv("TELEMETRY_MQTT_PASSWORD"); v != "" {
		c.Mosquitto.Password = v
	}

	if v := strings.TrimSpace(os.Getenv("TELEMETRY_RABBITMQ_URL")); v != "" {
		c.RabbitMQ.Endpoint = v
	}
	if v := os.Getenv("TELEMETRY_RABBITMQ_USERNAME"); v != "" {
		c.RabbitMQ.UserName = v
	}
	if v := os.Getenv("TELEMETRY_RABBITMQ_PASSWORD"); v != "" {
		c.RabbitMQ.Password = v
	}
	if v := strings.TrimSpace(os.Getenv("TELEMETRY_RABBITMQ_EXCHANGE")); v != "" {
		c.RabbitMQ.Exchange = v
	}
}
