package handlers

import (
	"html/template"
	"net/http"

	"biblioteca-betania/internal/config"
	"biblioteca-betania/internal/curator"
	"biblioteca-betania/internal/database"
)

type Handler struct {
	cfg       *config.Config
	db        *database.DB
	curator   *curator.Worker
	templates *template.Template
}

func NewHandler(cfg *config.Config, db *database.DB, curatorWorker *curator.Worker, tmpl *template.Template) *Handler {
	return &Handler{
		cfg:       cfg,
		db:        db,
		curator:   curatorWorker,
		templates: tmpl,
	}
}

// SecurityHeadersMiddleware injeta cabeçalhos de segurança contra XSS, Clickjacking e MIME sniffing
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com data:; img-src 'self' data: https:; frame-src 'self' https:;")
		next.ServeHTTP(w, r)
	})
}
