package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/sumanth/ai-governance-intake/internal/config"
	"github.com/sumanth/ai-governance-intake/internal/db"
	"github.com/sumanth/ai-governance-intake/internal/embeddings"
	"github.com/sumanth/ai-governance-intake/internal/llm"
	"github.com/sumanth/ai-governance-intake/internal/proposals"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	// Run migrations relative to the working directory (repo root)
	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("getwd: %v", err)
	}
	migrationsDir := filepath.Join(root, "migrations")

	if err := db.RunMigrations(ctx, pool, migrationsDir); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	embClient := embeddings.New(cfg.OpenRouterAPIKey, cfg.EmbedModel)
	llmClient := llm.New(cfg.OpenRouterAPIKey, cfg.LLMModel)

	repo := proposals.NewRepository(pool)
	svc := proposals.NewService(repo, embClient, llmClient, cfg)
	h := proposals.NewHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// API routes
	r.Post("/proposals", h.Submit)
	r.Get("/proposals", h.List)
	r.Get("/proposals/{id}", h.GetByID)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"degraded","db":"error"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","db":"ok"}`))
	})

	// Serve static files (frontend)
	staticDir := filepath.Join(root, "static")
	r.Handle("/*", http.FileServer(http.Dir(staticDir)))

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}
