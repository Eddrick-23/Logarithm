package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)

type Config struct {
	IngesterPort 	string
	AppPort 		string
	DBAddress 		string
	DBUser 			string
	DBPassword		string
	DBName 			string
	DBTableName		string
	NatsURL			string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil{
		slog.Info("No .env found. Reading directly and using fallbacks if needed.")
	}

	return &Config{
		IngesterPort: getEnv("INGESTER_PORT", "8090"),
		AppPort: getEnv("APP_PORT", "8091"),
		DBAddress: getEnv("DB_ADDRESS", "localhost:9000"),
		DBUser: getEnv("DB_USER", "admin"),
		DBPassword: getEnv("DB_PASSWORD", "strongpassword"),
		DBName: getEnv("DB_NAME", "logarithm"),
		DBTableName: getEnv("DB_TABLE_NAME", "logs"),
		NatsURL: getEnv("NATS_URL", nats.DefaultURL),
	}
}

func getEnv(key string, fallback string) string{
	value, exists:= os.LookupEnv(key)

	if (exists) {
		return value
	}
	return fallback
	
}
