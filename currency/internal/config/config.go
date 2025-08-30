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
	configPath := os.Getenv("CONFIG_PATH_CURRENCY")
	if configPath == "" {
		configPath = filepath.Join("config.example.yaml")
		logger.Info("CONFIG_PATH not set, using default path", zap.String("path", configPath))
	} else {
		logger.Info("Using config from ENV", zap.String("path", configPath))
	}
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		err = fmt.Errorf("failed to load the configuration file from disk: %w", err)
		log.Fatal("fatal in Read In Config\n", zap.Error(err))
	}

	config := &ConfigParam{
		CurrencyAPI: viper.GetString("currency_api_url"),
		GRPCPort:    viper.GetString("grpc_port"),
		HTTPPort:    viper.GetString("http_port"),
		ConnDB:      viper.GetString("connectDB"),
		NameDB:      viper.GetString("nameDB"),
		JWTToken:    viper.GetString("jwt_token"),
	}

	log.Println("Configuration loaded", zap.Any("config", config))

	return config, nil
}

type ConfigParam struct {
	CurrencyAPI string
	GRPCPort    string
	HTTPPort    string
	ConnDB      string
	NameDB      string
	JWTToken    string
}
