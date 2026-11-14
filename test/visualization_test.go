package test

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"biblioteca-betania/internal/config"
	"biblioteca-betania/internal/curator"
	"biblioteca-betania/internal/database"
	"biblioteca-betania/internal/handlers"
	"biblioteca-betania/web"
)

// TestVisualizationFullE2E valida a renderização visual completa de todas as páginas e do acervo angolano
func TestVisualizationFullE2E(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		UploadDir: t.TempDir(),
	}

	evaluator := curator.NewEvaluator(cfg)
	worker := curator.NewWorker(db, evaluator)
	tmpl, err := web.CarregarTemplates()
	if err != nil {
		t.Fatalf("erro ao carregar templates: %v", err)
	}

	h := handlers.NewHandler(cfg, db, worker, tmpl)

	// 1. Validar que o acervo angolano expandido foi carregado com sucesso no banco
	stats, err := db.ObterEstatisticas()
	if err != nil {
		t.Fatalf("erro ao obter estatísticas: %v", err)
	}
	if stats.TotalLivros < 120 {
		t.Fatalf("esperava acervo expandido com mais de 120 livros, encontrados: %d", stats.TotalLivros)
	}
	t.Logf("✓ Acervo Angolano Expandido verificado com sucesso: %d livros ativos catalogados", stats.TotalLivros)

	// 2. Testar Visualização da Página Inicial (Home / Catálogo)
	reqHome := httptest.NewRequest("GET", "/", nil)
	wHome := httptest.NewRecorder()
	h.HandleIndex(wHome, reqHome)

	if wHome.Code != 200 {
		t.Fatalf("esperava status 200 na home, recebido: %d. Body: %s", wHome.Code, wHome.Body.String())
	}
	bodyHome := wHome.Body.String()

	// Validações de elementos visuais do cabeçalho e marca
	elementosObrigatorios := []string{
		"BiblioTânia",
		"Colégio Betânia",
		"Acervo Aberto do Colégio Betânia",
		"search-box-card",
		"disciplines-pills-bar",
		"books-grid",
		"book-card",
		"btn-read",
		"stats-banner",
		"site-footer",
	}
	for _, el := range elementosObrigatorios {
		if !strings.Contains(bodyHome, el) {
			t.Fatalf("elemento visual obrigatório '%s' não encontrado no HTML da Home", el)
		}
	}

	// Verificar se as novas categorias angolanas aparecem nas pílulas de navegação
	pilulasEsperadas := []string{
		"Línguas Nacionais",
		"Agropecuária",
		"Saúde Comunitária",
		"Língua Portuguesa",
		"Matemática",
		"Física",
		"Química",
	}
	for _, p := range pilulasEsperadas {
		if !strings.Contains(bodyHome, p) {
			t.Fatalf("pílula de disciplina '%s' não encontrada na barra de navegação", p)
		}
	}
	t.Logf("✓ Pílulas visuais de disciplinas e navegação renderizadas perfeitamente")

	// 3. Testar Visualização da Paginação
	reqPag2 := httptest.NewRequest("GET", "/?p=2", nil)
	wPag2 := httptest.NewRecorder()
	h.HandleIndex(wPag2, reqPag2)

	if wPag2.Code != 200 {
		t.Fatalf("esperava status 200 na página 2, recebido: %d", wPag2.Code)
	}
	bodyPag2 := wPag2.Body.String()
	if !strings.Contains(bodyPag2, "pagination-nav") {
		t.Fatalf("barra visual de paginação deve estar presente")
	}
	if !strings.Contains(bodyPag2, "Página 2 de") {
		t.Fatalf("indicador de página 'Página 2 de' não encontrado")
	}
	if !strings.Contains(bodyPag2, "Anterior") || !strings.Contains(bodyPag2, "Próxima") {
		t.Fatalf("botões de navegação Anterior/Próxima devem estar visíveis na página 2")
	}
	t.Logf("✓ Barra de paginação visual testada e aprovada")

	// 4. Testar Visualização de Busca e Autores Angolanos
	autoresParaTestar := []struct {
		busca         string
		termoEsperado string
	}{
		{"Ondjaki", "Bom Dia Camaradas"},
		{"Agualusa", "Teoria Geral do Esquecimento"},
		{"Paula Tavares", "Ritos de Passagem"},
		{"Pepetela", "Mayombe"},
		{"Kimbundu", "Gramática Elementar da Língua Kimbundu"},
		{"Umbundu", "Dicionário Básico de Umbundu"},
		{"Mandioca", "Guia Prático da Mandioca"},
		{"Malária", "Prevenção e Tratamento da Malária"},
	}

	for _, tc := range autoresParaTestar {
		urlPath := fmt.Sprintf("/?q=%s", url.QueryEscape(tc.busca))
		reqBusca := httptest.NewRequest("GET", urlPath, nil)
		wBusca := httptest.NewRecorder()
		h.HandleIndex(wBusca, reqBusca)

		if wBusca.Code != 200 {
			t.Fatalf("falha na requisição de busca para '%s'", tc.busca)
		}
		bodyBusca := wBusca.Body.String()
		if !strings.Contains(bodyBusca, tc.termoEsperado) {
			t.Fatalf("busca por '%s' deveria renderizar o livro '%s'", tc.busca, tc.termoEsperado)
		}
		t.Logf("✓ Visualização da busca por '%s' exibiu com sucesso '%s'", tc.busca, tc.termoEsperado)
	}

	// 5. Testar Visualização do Formulário de Submissão de Alunos (/submeter)
	reqSubmeter := httptest.NewRequest("GET", "/submeter", nil)
	wSubmeter := httptest.NewRecorder()
	h.HandleSubmeterPagina(wSubmeter, reqSubmeter)

	if wSubmeter.Code != 200 {
		t.Fatalf("esperava 200 em /submeter, recebido: %d", wSubmeter.Code)
	}
	bodySubmeter := wSubmeter.Body.String()
	camposFormulario := []string{
		"form-card",
		"estudante_nome",
		"estudante_classe",
		"disciplina_id",
		"nivel_id",
		"tipo_envio",
		"grupo-arquivo",
		"grupo-link",
	}
	for _, campo := range camposFormulario {
		if !strings.Contains(bodySubmeter, campo) {
			t.Fatalf("campo ou container '%s' não encontrado no formulário de submissão", campo)
		}
	}
	t.Logf("✓ Formulário visual de submissão de estudantes validado")

	// 6. Testar Visualização do Mural da IA (/submissoes)
	reqMural := httptest.NewRequest("GET", "/submissoes", nil)
	wMural := httptest.NewRecorder()
	h.HandleSubmissoesStatus(wMural, reqMural)

	if wMural.Code != 200 {
		t.Fatalf("esperava 200 em /submissoes, recebido: %d", wMural.Code)
	}
	bodyMural := wMural.Body.String()
	if !strings.Contains(bodyMural, "Mural de Triagem da Inteligência Artificial") {
		t.Fatalf("título do mural da IA não renderizado")
	}
	t.Logf("✓ Mural visual da IA validado")

	// 7. Testar Visualização da Página Sobre (/sobre)
	reqSobre := httptest.NewRequest("GET", "/sobre", nil)
	wSobre := httptest.NewRecorder()
	h.HandleSobre(wSobre, reqSobre)

	if wSobre.Code != 200 {
		t.Fatalf("esperava 200 em /sobre, recebido: %d", wSobre.Code)
	}
	bodySobre := wSobre.Body.String()
	if !strings.Contains(bodySobre, "Sobre a BiblioTânia") || !strings.Contains(bodySobre, "Colégio Betânia") {
		t.Fatalf("conteúdo institucional do Colégio Betânia não renderizado em /sobre")
	}
	t.Logf("✓ Página visual 'Sobre' validada")

	// 8. Testar Redirecionamento e Leitura de Livro (/ler/:id)
	livros, total, _ := db.ListarLivros(database.FiltrosBusca{Query: "Mayombe"})
	if total == 0 || len(livros) == 0 {
		t.Fatalf("esperava encontrar Mayombe")
	}
	mayombeID := livros[0].ID

	reqLer := httptest.NewRequest("GET", fmt.Sprintf("/ler/%d", mayombeID), nil)
	wLer := httptest.NewRecorder()
	h.HandleLerLivro(wLer, reqLer)

	if wLer.Code != 302 {
		t.Fatalf("esperava redirecionamento 302 para leitura externa, recebido: %d", wLer.Code)
	}
	loc := wLer.Header().Get("Location")
	if !strings.Contains(loc, "ueangola.com") {
		t.Fatalf("redirecionamento de Mayombe deveria apontar para o acervo oficial da UEA, recebido: %s", loc)
	}
	t.Logf("✓ Redirecionamento seguro de leitura validado para: %s", loc)

	// 9. Testar Seletor Rápido de Classes Escolares e Filtragem Curricular (1ª à 12ª Classe)
	if !strings.Contains(bodyHome, "class-selector-card") {
		t.Fatalf("seletor visual de classes escolares não encontrado na Home")
	}
	if !strings.Contains(bodyHome, "Selecione a sua Classe Escolar") {
		t.Fatalf("título do seletor de classes escolares ausente na Home")
	}

	// Validar que TODAS as classes de 1ª a 12ª possuem manuais oficiais no banco e filtram perfeitamente
	for c := 1; c <= 12; c++ {
		reqClasse := httptest.NewRequest("GET", fmt.Sprintf("/?classe=%d", c), nil)
		wClasse := httptest.NewRecorder()
		h.HandleIndex(wClasse, reqClasse)

		if wClasse.Code != 200 {
			t.Fatalf("esperava status 200 ao filtrar classe %d, recebido: %d", c, wClasse.Code)
		}
		bodyClasse := wClasse.Body.String()
		badgeEsperado := fmt.Sprintf("🎒 %dª Classe", c)
		if !strings.Contains(bodyClasse, badgeEsperado) {
			t.Fatalf("badge visual de classe '%s' não encontrado ao acessar /?classe=%d", badgeEsperado, c)
		}
		if !strings.Contains(bodyClasse, "📘 Manual Oficial") {
			t.Fatalf("badge '📘 Manual Oficial' não encontrado na visualização da %dª classe", c)
		}

		livrosClasse, totalClasse, err := db.ListarLivros(database.FiltrosBusca{Classe: c})
		if err != nil {
			t.Fatalf("erro ao consultar livros da %dª classe: %v", c, err)
		}
		if totalClasse == 0 || len(livrosClasse) == 0 {
			t.Fatalf("a %dª classe deve possuir manuais escolares no acervo, encontrados 0", c)
		}
		t.Logf("✓ Classe %dª validada: %d obras oficiais catalogadas com sucesso", c, totalClasse)
	}

	// 10. Testar Presença de Capas Reais 3:4 e Efeito de Lombada Física
	if !strings.Contains(bodyHome, "book-cover-container") || !strings.Contains(bodyHome, "book-cover-img") {
		t.Fatalf("elementos de capa de livro não encontrados no HTML da Home")
	}
	if !strings.Contains(bodyHome, "book-cover-spine") {
		t.Fatalf("efeito físico de lombada da capa ausente no HTML")
	}
	t.Logf("✓ Capas editoriais 3:4 e lombadas renderizadas com sucesso nos cartões")
}
