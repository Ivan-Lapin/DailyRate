package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func LoadConfig(configPath string, logger *zap.Logger) (*ConfigParam, error) {

	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		err = fmt.Errorf("failed to set config file: %w", err)
		log.Fatal(err)
	}

	config := &ConfigParam{
		CurrencyAPI: viper.GetString("currency_api_url"),
		GRPCPort:    viper.GetString("grpc_port"),
		HTTPPort:    viper.GetString("http_port"),
	}

	logger.Info("Configuration loaded", zap.Any("config", config))

	return config, nil
}

type ConfigParam struct {
	CurrencyAPI string
	GRPCPort    string
	HTTPPort    string
}
