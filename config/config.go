package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func LoadConfig(target any, path string) error {
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "dev"
	}
	fmt.Printf("🔧 Loading configuration for environment: %s\n", env)

	v := viper.New()
	v.SetConfigName(env)
	v.SetConfigType("yaml")

	if path != "" {
		v.AddConfigPath(path)
	} else {
		v.AddConfigPath("./")
	}

	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		absPath, _ := filepath.Abs(filepath.Join(path, env+".yaml"))
		return fmt.Errorf("❌ failed to read config file (%s): %w", absPath, err)
	}

	if err := v.Unmarshal(target); err != nil {
		return fmt.Errorf("❌ unable to unmarshal config into struct: %w", err)
	}

	fmt.Println("✅ Configuration loaded successfully!")
	return nil
}
