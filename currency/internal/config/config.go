package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func LoadConfig(logger *zap.Logger) (*ConfigParam, error) {
	exePath, err := os.Executable()
	if err != nil {
		err := fmt.Errorf("cannot get executable path: %w", err)
		log.Fatal(err)
	}
	configPath := filepath.Join(filepath.Dir(exePath), "configs", "config.example.yaml")
	fmt.Println(configPath)
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		err = fmt.Errorf("failed to set config file: %w", err)
		log.Fatal(err)
	}

	config := &ConfigParam{
		CurrencyAPI: viper.GetString("currency_api_url"),
		GRPCPort:    viper.GetString("grpc_port"),
		HTTPPort:    viper.GetString("http_port"),
		ConnDB:      viper.GetString("connectDB"),
		NameDB:      viper.GetString("nameDB"),
	}

	logger.Info("Configuration loaded", zap.Any("config", config))

	return config, nil
}

type ConfigParam struct {
	CurrencyAPI string
	GRPCPort    string
	HTTPPort    string
	ConnDB      string
	NameDB      string
}
