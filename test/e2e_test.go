package test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"biblioteca-betania/internal/config"
	"biblioteca-betania/internal/curator"
	"biblioteca-betania/internal/database"
	"biblioteca-betania/internal/handlers"
	"biblioteca-betania/internal/models"
	"biblioteca-betania/web"
)

func TestFullStudentSubmissionAndAutoCatalogWorkflow(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		UploadDir:   t.TempDir(),
		MaxFileSize: 50 * 1024 * 1024,
	}

	evaluator := curator.NewEvaluator(cfg)
	worker := curator.NewWorker(db, evaluator)
	worker.Start()
	defer worker.Stop()

	tmpl, err := web.CarregarTemplates()
	if err != nil {
		t.Fatalf("erro ao carregar templates: %v", err)
	}

	h := handlers.NewHandler(cfg, db, worker, tmpl)

	// 1. Verificar Home inicial
	reqIndex := httptest.NewRequest("GET", "/", nil)
	wIndex := httptest.NewRecorder()
	h.HandleIndex(wIndex, reqIndex)

	if wIndex.Code != 200 {
		t.Fatalf("esperava status 200 na home, recebido: %d", wIndex.Code)
	}
	bodyIndex := wIndex.Body.String()
	if !strings.Contains(bodyIndex, "BiblioTânia") {
		t.Fatalf("home deve conter o título BiblioTânia")
	}
	if !strings.Contains(bodyIndex, "Colégio Betânia") {
		t.Fatalf("home deve referenciar o Colégio Betânia")
	}

	// 2. Estudante envia um livro em formato Link
	var b bytes.Buffer
	wMultipart := multipart.NewWriter(&b)
	_ = wMultipart.WriteField("estudante_nome", "Tânia da Costa Mendes")
	_ = wMultipart.WriteField("estudante_classe", "12ª Classe - Informática")
	_ = wMultipart.WriteField("titulo", "Estruturas de Dados e Algoritmos em Go")
	_ = wMultipart.WriteField("autor", "Dra. Esperança Manuel")
	_ = wMultipart.WriteField("tipo_envio", "link")
	_ = wMultipart.WriteField("link_url", "https://openlibrary.org/works/OL123456W")
	_ = wMultipart.WriteField("disciplina_id", "9") // Informática
	_ = wMultipart.WriteField("nivel_id", "3")      // II Ciclo
	_ = wMultipart.Close()

	reqSub := httptest.NewRequest("POST", "/api/submissoes", &b)
	reqSub.Header.Set("Content-Type", wMultipart.FormDataContentType())
	wSub := httptest.NewRecorder()

	h.HandleProcessarSubmissao(wSub, reqSub)

	if wSub.Code != 303 {
		t.Fatalf("esperava redirecionamento 303 após submissão, recebido: %d", wSub.Code)
	}

	// 3. Aguardar brevemente o Worker de IA processar a submissão
	time.Sleep(300 * time.Millisecond)

	// 4. Verificar se a submissão foi aprovada
	submissoes, err := db.ListarSubmissoesRecentes(10)
	if err != nil || len(submissoes) == 0 {
		t.Fatalf("esperava submissão registrada no banco")
	}

	sub := submissoes[0]
	if sub.Status != "aprovado" {
		t.Fatalf("esperava status 'aprovado' para livro legítimo, recebido: %s (parecer: %s)", sub.Status, sub.AIParecer)
	}

	// 5. Verificar se o livro foi adicionado automaticamente ao catálogo ativo
	livros, total, err := db.ListarLivros(database.FiltrosBusca{Query: "Estruturas de Dados"})
	if err != nil {
		t.Fatalf("erro ao consultar catálogo: %v", err)
	}
	if total == 0 || len(livros) == 0 {
		t.Fatalf("livro aprovado pela IA deveria estar presente na busca do catálogo!")
	}
	if !strings.Contains(livros[0].Titulo, "Estruturas de Dados") {
		t.Fatalf("título inesperado: %s", livros[0].Titulo)
	}

	// 6. Testar endpoint de estatísticas
	reqStats := httptest.NewRequest("GET", "/api/stats", nil)
	wStats := httptest.NewRecorder()
	h.HandleAPIStats(wStats, reqStats)

	if wStats.Code != 200 {
		t.Fatalf("esperava 200 em /api/stats")
	}
	var stats models.EstatisticasAcervo
	if err := json.Unmarshal(wStats.Body.Bytes(), &stats); err != nil {
		t.Fatalf("resposta /api/stats deve ser json válido: %v", err)
	}
	if stats.TotalAprovadosIA == 0 {
		t.Fatalf("total de aprovados da IA deveria ser >= 1")
	}
}
