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

// SecurityHeadersMiddleware injeta cabeçalhos de segurança contra XSS, Clickjacking, MIME sniffing, HSTS e suporte a CORS para o GitHub Pages
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		w.Header().Set("Permissions-Policy", "geolocation=(), camera=(), microphone=(), payment=()")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept")
		w.Header().Set("Content-Security-Policy", "default-src 'self' https://misterantoy.github.io; script-src 'self' https://misterantoy.github.io; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com data:; img-src 'self' data: https:; frame-src 'self' https:; connect-src 'self' https:;")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
