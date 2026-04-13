package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           int
	ListenHost     string
	PublicBaseURL  string
	APIBaseURL     string
	APIKey         string
	Model          string
	RequestTimeout time.Duration
	ArtifactDir    string
}

func loadConfig() Config {
	_ = godotenv.Load(".env.local")
	_ = godotenv.Load(".env")

	port := getEnvInt("CCBOS_MCP_PORT", 18191)
	requestTimeoutSeconds := getEnvInt("CCBOS_REQUEST_TIMEOUT_SECONDS", 60)
	artifactDir := os.Getenv("CCBOS_ARTIFACT_DIR")
	if artifactDir == "" {
		artifactDir = filepath.Join("artifacts")
	}

	listenHost := getEnv("CCBOS_LISTEN_HOST", "127.0.0.1")
	publicBaseURL := strings.TrimSpace(os.Getenv("CCBOS_PUBLIC_BASE_URL"))
	if publicBaseURL == "" {
		publicBaseURL = "http://" + listenHost + ":" + strconv.Itoa(port)
	}

	return Config{
		Port:           port,
		ListenHost:     listenHost,
		PublicBaseURL:  publicBaseURL,
		APIBaseURL:     getEnv("CCBOS_API_BASE_URL", "https://api.deepseek.com/v1"),
		APIKey:         os.Getenv("CCBOS_API_KEY"),
		Model:          getEnv("CCBOS_MODEL", "deepseek-chat"),
		RequestTimeout: time.Duration(requestTimeoutSeconds) * time.Second,
		ArtifactDir:    artifactDir,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}
