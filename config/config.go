package config

import (
	"fmt"
	"os"
)

type Config struct {
	ServerAddr       string
	Port             string
	WSServerAddr     string
	WSPort           string
	MikrotikAddress  string
	MikrotikPort     string
	MikrotikUser     string
	MikrotikPassword string
	DatabaseDSN      string
}

func LoadConfig() *Config {
	// Load from environment or use defaults
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPass := getEnv("DB_PASS", "r00t")
	dbName := getEnv("DB_NAME", "mikrobill")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local",
		dbUser, dbPass, dbHost, dbPort, dbName)

	return &Config{
		ServerAddr:       getEnv("SERVER_ADDR", ":8080"),
		Port:             getEnv("PORT", "8080"),
		WSServerAddr:     getEnv("WS_SERVER_ADDR", ":8081"),
		WSPort:           getEnv("WS_PORT", "8081"),
		MikrotikAddress:  getEnv("MIKROTIK_HOST", "192.168.1.1"),
		MikrotikPort:     getEnv("MIKROTIK_PORT", "8728"),
		MikrotikUser:     getEnv("MIKROTIK_USER", "admin"),
		MikrotikPassword: getEnv("MIKROTIK_PASS", "password"),
		DatabaseDSN:      dsn,
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}