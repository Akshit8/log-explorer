package main

import (
	"fmt"

	"github.com/spf13/viper"
)

type ServiceConfig struct {
	Name      string `mapstructure:"name"`
	AccountID string `mapstructure:"accountId"`
	APIToken  string `mapstructure:"apiToken"`
	Env       *string `mapstructure:"env"`
}

type Config struct {
	Services []ServiceConfig `mapstructure:"services"`
}

func (c *Config) Validate() error {
	for i, svc := range c.Services {
		if svc.Name == "" || svc.AccountID == "" || svc.APIToken == "" {
			return fmt.Errorf("service at index %d is missing required fields (name, accountId, or apiToken)", i)
		}
	}
	return nil
}

func LoadConfig() (*Config, error) {
	v := viper.New()
	
	// Configuration lookups
	v.SetConfigName("config") // name of config file (without extension)
	v.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	v.AddConfigPath(".")      // look for config in the working directory

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}