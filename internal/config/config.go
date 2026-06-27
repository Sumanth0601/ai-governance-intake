package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL         string
	OpenRouterAPIKey    string
	EmbedModel          string
	LLMModel            string
	Port                string
	SimilarityThreshold float64
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY is required")
	}

	embedModel := os.Getenv("OPENROUTER_EMBED_MODEL")
	if embedModel == "" {
		embedModel = "nvidia/llama-nemotron-embed-vl-1b-v2:free"
	}

	llmModel := os.Getenv("OPENROUTER_MODEL")
	if llmModel == "" {
		llmModel = "openai/gpt-oss-120b:free"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	threshold := 0.85
	if raw := os.Getenv("SIMILARITY_THRESHOLD"); raw != "" {
		var err error
		threshold, err = strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid SIMILARITY_THRESHOLD: %w", err)
		}
	}

	return &Config{
		DatabaseURL:         dbURL,
		OpenRouterAPIKey:    apiKey,
		EmbedModel:          embedModel,
		LLMModel:            llmModel,
		Port:                port,
		SimilarityThreshold: threshold,
	}, nil
}
