package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)

type Config struct {
	IngesterHost    string
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
	if err := godotenv.Load(".env.local", ".env"); err != nil{
		fmt.Println("Note: No .env found. using system environemnt variables with default fallbacks if needed.")
	}

	return &Config{
		IngesterHost: getEnv("INGESTER_HOST", "localhost"),
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
