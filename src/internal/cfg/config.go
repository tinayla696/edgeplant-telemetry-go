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
	Gpsd      GpsdCfg                 `json:"gpsd" yaml:"gpsd"`
}

// SocketCAN configuration struct
type SocketCanCfg struct {
	Type    string `json:"type" yaml:"type"`
	Bitrate uint32 `json:"bitrate" yaml:"bitrate"`
	DbcPath string `json:"dbc_path" yaml:"dbc_path"`
}

// MQTT configuration struct
type MqttCfg struct {
	Endpoint   string `json:"endpoint" yaml:"endpoint"`
	UserName   string `json:"username" yaml:"username"`
	Password   string `json:"password" yaml:"password"`
	UseTLS     bool   `json:"use_tls" yaml:"use_tls"`
	CertPath   string `json:"ca_cert_path" yaml:"ca_cert_path"`
	ClientCert string `json:"client_cert" yaml:"client_cert"`
	ClientKey  string `json:"client_key" yaml:"client_key"`
}

// GPSD configuration struct
type GpsdCfg struct {
	Endpoint string `json:"endpoint" yaml:"endpoint"`
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

	return &cfg, nil
}
