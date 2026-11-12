package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port         int
	DBPath       string
	UploadDir    string
	MaxFileSize  int64  // in bytes
	GeminiAPIKey string
	OpenAIAPIKey string
	OpenAIBaseURL string
	AdminToken   string
}

func Load() *Config {
	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/bibliotania.db"
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	maxFileSize := int64(50 * 1024 * 1024) // 50MB default
	if m := os.Getenv("MAX_FILE_SIZE_MB"); m != "" {
		if v, err := strconv.ParseInt(m, 10, 64); err == nil && v > 0 {
			maxFileSize = v * 1024 * 1024
		}
	}

	adminToken := os.Getenv("ADMIN_TOKEN")
	if adminToken == "" {
		adminToken = "betania-admin-2026"
	}

	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")
	if openAIBaseURL == "" {
		openAIBaseURL = "https://api.openai.com/v1"
	}

	return &Config{
		Port:          port,
		DBPath:        dbPath,
		UploadDir:     uploadDir,
		MaxFileSize:   maxFileSize,
		GeminiAPIKey:  os.Getenv("GEMINI_API_KEY"),
		OpenAIAPIKey:  os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL: openAIBaseURL,
		AdminToken:    adminToken,
	}
}
