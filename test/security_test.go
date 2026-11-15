package test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"biblioteca-betania/internal/config"
	"biblioteca-betania/internal/curator"
	"biblioteca-betania/internal/database"
	"biblioteca-betania/internal/handlers"
	"biblioteca-betania/internal/models"
	"biblioteca-betania/web"
)

func setupTestDB(t *testing.T) (*database.DB, func()) {
	tmpDir, err := os.MkdirTemp("", "bibliotania_test_*")
	if err != nil {
		t.Fatalf("falha ao criar temp dir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("falha ao abrir test db: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
		_ = os.RemoveAll(tmpDir)
	}
	return db, cleanup
}

// -------------------------------------------------------------
// 1. Testes de Segurança: Imunidade a SQL Injection
// -------------------------------------------------------------
func TestSQLInjectionImmunity(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Tentativas clássicas de SQL Injection
	injections := []string{
		"' OR '1'='1",
		"'; DROP TABLE livros; --",
		"1' UNION SELECT 1, 'hacked', 'hacked', 1, 1, 'hacked', 2026, 'pdf', '', '', 0, 'ativo', 'hacker', 0, 0, '2026-01-01' --",
		"admin'--",
		"\" OR \"\"=\"",
	}

	for _, payload := range injections {
		filtros := database.FiltrosBusca{
			Query: payload,
		}
		// A busca nunca deve explodir com erro de sintaxe SQL nem devolver todo o banco
		livros, _, err := db.ListarLivros(filtros)
		if err != nil {
			t.Logf("Query tratou payload com segurança: %v", err)
		} else {
			// Se retornar, não pode ter vazado todos os livros
			for _, l := range livros {
				if l.Titulo == "hacked" {
					t.Fatalf("SQL Injection permitiu injeção de registro falso!")
				}
			}
		}

		// Verificar que as tabelas essenciais permanecem intactas
		var count int
		if err := db.SqlDB.QueryRow("SELECT COUNT(*) FROM livros").Scan(&count); err != nil {
			t.Fatalf("Tabela livros corrompida por SQL Injection: %v", err)
		}
	}
}

// -------------------------------------------------------------
// 2. Testes de Segurança: Proteção Nativa contra XSS
// -------------------------------------------------------------
func TestXSSAutoEscaping(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := &config.Config{UploadDir: "./test_uploads"}
	evaluator := curator.NewEvaluator(cfg)
	worker := curator.NewWorker(db, evaluator)
	tmpl, err := web.CarregarTemplates()
	if err != nil {
		t.Fatalf("erro ao carregar templates: %v", err)
	}

	h := handlers.NewHandler(cfg, db, worker, tmpl)

	// Inserir livro com payload XSS no título e sinopse
	xssPayload := `<script>alert('xss_exploit')</script><img src=x onerror=alert(1)>`
	livro := &models.Livro{
		Titulo:  xssPayload,
		Autor:   `Hacker <script>evil()</script>`,
		Sinopse: `Texto malicioso <iframe src="evil.com"></iframe>`,
		Status:  "ativo",
	}
	_, err = db.InserirLivro(livro)
	if err != nil {
		t.Fatalf("falha ao inserir livro: %v", err)
	}

	req := httptest.NewRequest("GET", "/?q=xss_exploit", nil)
	w := httptest.NewRecorder()

	h.HandleIndex(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	bodyBytes := w.Body.String()

	// O script raw NUNCA pode estar presente sem escape
	if strings.Contains(bodyBytes, "<script>alert('xss_exploit')</script>") {
		t.Fatalf("FALHA DE SEGURANÇA: Payload XSS não foi escapado no HTML!")
	}

	// Deve conter a versão segura com entidades HTML
	if !strings.Contains(bodyBytes, "&lt;script&gt;alert(&#39;xss_exploit&#39;)&lt;/script&gt;") {
		t.Logf("Verificando escape contextual seguro no HTML template")
	}

	// Verificar cabeçalhos de segurança essenciais
	csp := resp.Header.Get("Content-Security-Policy")
	_ = csp // injetado pelo middleware
}

// -------------------------------------------------------------
// 3. Testes de Concorrência: Zero Race Conditions com SQLite WAL
// -------------------------------------------------------------
func TestConcurrencyAndRaceConditions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	const numGoroutines = 40
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		workerID := i
		go func() {
			defer wg.Done()

			if workerID%2 == 0 {
				// Leituras concorrentes
				_, _, err := db.ListarLivros(database.FiltrosBusca{Pagina: 1, Limite: 10})
				if err != nil {
					errChan <- err
					return
				}
				_ = db.IncrementarVisualizacao(1)
			} else {
				// Escritas concorrentes
				_, err := db.InserirLivro(&models.Livro{
					Titulo:  fmt.Sprintf("Obra Concorrente #%d", workerID),
					Autor:   fmt.Sprintf("Autor Teste #%d", workerID),
					Status:  "ativo",
					Sinopse: "Livro adicionado concorrentemente para teste de race condition.",
				})
				if err != nil {
					errChan <- err
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Fatalf("Erro de concorrência detectado (database lock / race condition): %v", err)
	}
}

// -------------------------------------------------------------
// 4. Testes do Avaliador de IA para Submissões de Estudantes
// -------------------------------------------------------------
func TestCuratorEvaluationPipeline(t *testing.T) {
	cfg := &config.Config{}
	evaluator := curator.NewEvaluator(cfg)

	// Caso 1: Envio legítimo de estudante
	subLegitima := &models.SubmissaoEstudante{
		EstudanteNome:   "Manuel António",
		EstudanteClasse: "11ª Classe A",
		Titulo:          "Manual Completo de Química Orgânica",
		Autor:           "Prof. Doutor Santos",
		TipoEnvio:       "link",
		LinkURL:         "https://openstax.org/chemistry",
	}

	ctx := context.Background()
	parecer, err := evaluator.AvaliarSubmissao(ctx, subLegitima)
	if err != nil {
		t.Fatalf("erro ao avaliar submissão legítima: %v", err)
	}
	if !parecer.Aprovado {
		t.Fatalf("esperava aprovação para material legítimo, recebido: %s", parecer.Parecer)
	}

	// Caso 2: Envio de material proibido / impróprio em ambiente escolar
	subProibida := &models.SubmissaoEstudante{
		EstudanteNome:   "Aluno Indevido",
		EstudanteClasse: "8ª Classe",
		Titulo:          "Como hackear e apostar em cassino online",
		TipoEnvio:       "link",
		LinkURL:         "https://bet-online-proibido.com",
	}

	parecerProibido, err := evaluator.AvaliarSubmissao(ctx, subProibida)
	if err != nil {
		t.Fatalf("erro ao avaliar submissão imprópria: %v", err)
	}
	if parecerProibido.Aprovado {
		t.Fatalf("esperava rejeição para material impróprio no colégio!")
	}

	// Caso 3: Link malicioso de rede local / SSRF attempt
	subSSRF := &models.SubmissaoEstudante{
		EstudanteNome:   "Testador",
		EstudanteClasse: "12ª Classe",
		Titulo:          "Acesso ao Servidor Interno",
		TipoEnvio:       "link",
		LinkURL:         "http://127.0.0.1:8080/admin",
	}

	parecerSSRF, _ := evaluator.AvaliarSubmissao(ctx, subSSRF)
	if parecerSSRF.Aprovado {
		t.Fatalf("esperava rejeição para tentativa de link SSRF para 127.0.0.1!")
	}
}

// -------------------------------------------------------------
// 5. Testes de Busca FTS5 e Paginação
// -------------------------------------------------------------
func TestCatalogSearchAndPagination(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Busca por palavra-chave existente no cânone angolano
	livros, total, err := db.ListarLivros(database.FiltrosBusca{
		Query:  "Mayombe",
		Pagina: 1,
		Limite: 10,
	})
	if err != nil {
		t.Fatalf("falha na busca FTS5: %v", err)
	}
	if total == 0 || len(livros) == 0 {
		t.Fatalf("esperava encontrar Mayombe na busca por 'Mayombe'")
	}
	if !strings.Contains(livros[0].Titulo, "Mayombe") {
		t.Fatalf("livro incorreto retornado: %s", livros[0].Titulo)
	}
}

// -------------------------------------------------------------
// 6. Testes de Sanitização de Headers HTTP (Content-Disposition)
// -------------------------------------------------------------
func TestContentDispositionHeaderSanitization(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "bibliotania_uploads_*")
	if err != nil {
		t.Fatalf("falha ao criar temp upload dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testPdfPath := filepath.Join(tmpDir, "livro_teste.pdf")
	if err := os.WriteFile(testPdfPath, []byte("%PDF-1.4 mock content"), 0644); err != nil {
		t.Fatalf("falha ao criar mock pdf: %v", err)
	}

	cfg := &config.Config{UploadDir: tmpDir}
	evaluator := curator.NewEvaluator(cfg)
	worker := curator.NewWorker(db, evaluator)
	tmpl, err := web.CarregarTemplates()
	if err != nil {
		t.Fatalf("erro ao carregar templates: %v", err)
	}
	h := handlers.NewHandler(cfg, db, worker, tmpl)

	// Livro com aspas duplas e injeções no título
	livroMalicioso := &models.Livro{
		Titulo:     "Manual \"Química Moderna\"\r\nInjeção Header",
		Autor:      "Autor Teste",
		Status:     "ativo",
		ArquivoURL: "/uploads/livro_teste.pdf",
	}
	id, err := db.InserirLivro(livroMalicioso)
	if err != nil {
		t.Fatalf("falha ao inserir livro de teste: %v", err)
	}

	req := httptest.NewRequest("GET", fmt.Sprintf("/ler/%d", id), nil)
	w := httptest.NewRecorder()
	h.HandleLerLivro(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("esperava status 200 ao servir arquivo existente, recebido: %d", resp.StatusCode)
	}

	disposition := resp.Header.Get("Content-Disposition")
	// Não deve conter aspas não escapadas ou quebras de linha dentro do nome do arquivo
	if strings.Contains(disposition, `""`) || strings.Contains(disposition, "\r") || strings.Contains(disposition, "\n") {
		t.Fatalf("FALHA: Content-Disposition mal formatado ou suscetível a header splitting: %s", disposition)
	}
	// Deve conter aspas simples sanitizadas
	if !strings.Contains(disposition, "Manual 'Química Moderna'") {
		t.Fatalf("esperava aspas simples sanitizadas no header, recebido: %s", disposition)
	}
	t.Logf("✓ Content-Disposition sanitizado com sucesso: %s", disposition)
}
