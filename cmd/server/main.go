package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"biblioteca-betania/internal/config"
	"biblioteca-betania/internal/curator"
	"biblioteca-betania/internal/database"
	"biblioteca-betania/internal/handlers"
	"biblioteca-betania/web"
)

func main() {
	log.Println("[BiblioTânia] Iniciando servidor do Colégio Betânia")

	// 1. Carregar Configurações
	cfg := config.Load()

	// 2. Conectar ao Banco de Dados SQLite FTS5 Seguro
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("Erro fatal ao inicializar banco: %v", err)
	}
	defer db.Close()
	log.Printf("[DB] Banco SQLite carregado em: %s", cfg.DBPath)

	// 3. Carregar Templates HTML Embutidos
	tmpl, err := web.CarregarTemplates()
	if err != nil {
		log.Fatalf("Erro fatal ao carregar templates: %v", err)
	}

	// 4. Iniciar Avaliador de IA e Fila de Curadoria em Segundo Plano
	evaluator := curator.NewEvaluator(cfg)
	curatorWorker := curator.NewWorker(db, evaluator)
	curatorWorker.Start()
	defer curatorWorker.Stop()
	log.Println("[Curator] Worker de Triagem da IA iniciado.")

	// 5. Configurar Handlers e Rotas HTTP
	h := handlers.NewHandler(cfg, db, curatorWorker, tmpl)

	mux := http.NewServeMux()

	// Assets estáticos embutidos (CSS, JS, Favicon)
	mux.Handle("/static/", http.StripPrefix("/static/", web.StaticFileServer()))

	// Páginas Web
	mux.HandleFunc("/", h.HandleIndex)
	mux.HandleFunc("/livro/", h.HandleLivroDetalhes)
	mux.HandleFunc("/ler/", h.HandleLerLivro)
	mux.HandleFunc("/submeter", h.HandleSubmeterPagina)
	mux.HandleFunc("/submissoes", h.HandleSubmissoesStatus)
	mux.HandleFunc("/sobre", h.HandleSobre)

	// Endpoints de API e Ação
	mux.HandleFunc("/api/submissoes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.HandleAPISubmissoes(w, r)
			return
		}
		h.HandleProcessarSubmissao(w, r)
	})
	mux.HandleFunc("/api/livros", h.HandleAPILivros)
	mux.HandleFunc("/api/stats", h.HandleAPIStats)

	// Middleware de Segurança Global (Anti-XSS, Anti-Sniffing, CSP)
	secureHandler := handlers.SecurityHeadersMiddleware(mux)

	// 6. Inicializar Servidor HTTP com Timeouts Rigorosos
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      secureHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("[HTTP] Servidor ativo em http://localhost%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erro no servidor HTTP: %v", err)
		}
	}()

	// 7. Encerramento Gracioso (Graceful Shutdown)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("\n[HTTP] Encerrando BiblioTânia de forma segura...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Erro no encerramento: %v", err)
	}
	log.Println("[HTTP] Servidor finalizado com sucesso.")
}
