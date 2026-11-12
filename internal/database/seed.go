package database

import (
	"database/sql"
	"fmt"
)

// SeedInitialData popula as disciplinas do Colégio Betânia e livros fundamentais
func (db *DB) SeedInitialData() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// 1. Disciplinas
	disciplinas := []struct {
		nome  string
		slug  string
		icone string
		ordem int
	}{
		{"Língua Portuguesa & Literatura", "portugues-literatura", "📖", 1},
		{"Matemática & Geometria", "matematica", "📐", 2},
		{"Física", "fisica", "⚡", 3},
		{"Química", "quimica", "🧪", 4},
		{"Biologia & Ciências da Natureza", "biologia", "🔬", 5},
		{"História & Estudos Sociais", "historia", "🏛️", 6},
		{"Geografia", "geografia", "🌍", 7},
		{"Filosofia & Ética", "filosofia", "🧠", 8},
		{"Informática & Programação", "informatica", "💻", 9},
		{"Inglês & Línguas Estrangeiras", "ingles", "🌐", 10},
		{"Cultura Geral & Enciclopédias", "cultura-geral", "✨", 11},
	}

	for _, d := range disciplinas {
		_, err := db.SqlDB.Exec(`
			INSERT OR IGNORE INTO disciplinas (nome, slug, icone, ordem)
			VALUES (?, ?, ?, ?)
		`, d.nome, d.slug, d.icone, d.ordem)
		if err != nil {
			return fmt.Errorf("seed disciplina %s: %w", d.nome, err)
		}
	}

	// 2. Níveis de Ensino
	niveis := []struct {
		nome string
		slug string
	}{
		{"Ensino Primário (1ª à 6ª Classe)", "primario"},
		{"I Ciclo do Secundário (7ª à 9ª Classe)", "primeiro-ciclo"},
		{"II Ciclo / Pré-Universitário (10ª à 12ª/13ª Classe)", "segundo-ciclo"},
		{"Acervo Geral & Formação Pedagógica", "geral"},
	}

	for _, n := range niveis {
		_, err := db.SqlDB.Exec(`
			INSERT OR IGNORE INTO niveis (nome, slug)
			VALUES (?, ?)
		`, n.nome, n.slug)
		if err != nil {
			return fmt.Errorf("seed nivel %s: %w", n.nome, err)
		}
	}

	// 3. Livros Fundamentais Iniciais
	var count int
	_ = db.SqlDB.QueryRow("SELECT COUNT(*) FROM livros").Scan(&count)
	if count > 0 {
		return nil // já semeado
	}

	obras := []struct {
		titulo       string
		autor        string
		materiaSlug  string
		nivelSlug    string
		sinopse      string
		ano          int
		formato      string
		link         string
		submetidoPor string
	}{
		{
			"Dom Casmurro",
			"Machado de Assis",
			"portugues-literatura",
			"segundo-ciclo",
			"Um dos maiores clássicos da literatura em língua portuguesa. Bento Santiago narra suas memórias e seu amor por Capitu, abordando temas de ciúme, dúvida e psicologia humana.",
			1899,
			"link",
			"https://www.dominiopublico.gov.br/download/texto/bv000240.pdf",
			"Curadoria Colégio Betânia",
		},
		{
			"Os Lusíadas",
			"Luís de Camões",
			"portugues-literatura",
			"segundo-ciclo",
			"A epopeia clássica em dez cantos que celebra as navegações e a história lusófona no Renascimento.",
			1572,
			"link",
			"https://www.dominiopublico.gov.br/download/texto/bv000045.pdf",
			"Curadoria Colégio Betânia",
		},
		{
			"Fundamentos da Álgebra Elementar e Funções",
			"Coleção Didática Aberta",
			"matematica",
			"primeiro-ciclo",
			"Guia estruturado de matemática cobrindo polinômios, equações de 1º e 2º grau, frações e geometria plana para estudantes do ensino secundário.",
			2024,
			"link",
			"https://openstax.org/details/books/elementary-algebra-2e",
			"Curadoria Colégio Betânia",
		},
		{
			"Princípios de Física Mecânica e Termodinâmica",
			"OpenStax University Physics",
			"fisica",
			"segundo-ciclo",
			"Manual aberto internacional de física para estudantes pré-universitários, cobrindo vetores, cinemática, leis de Newton e energia.",
			2023,
			"link",
			"https://openstax.org/details/books/university-physics-volume-1",
			"Curadoria Colégio Betânia",
		},
		{
			"Química Geral e Estrutura Atómica",
			"OpenStax Chemistry",
			"quimica",
			"segundo-ciclo",
			"Conceitos de tabela periódica, ligações químicas, estequiometria e soluções com exercícios resolvidos.",
			2023,
			"link",
			"https://openstax.org/details/books/chemistry-2e",
			"Curadoria Colégio Betânia",
		},
		{
			"Biologia Celular, Genética e Ecologia",
			"OpenStax Biology",
			"biologia",
			"segundo-ciclo",
			"Aborda os fundamentos da vida, mitose e meiose, hereditariedade mendeliana e preservação dos ecossistemas africanos e mundiais.",
			2024,
			"link",
			"https://openstax.org/details/books/biology-2e",
			"Curadoria Colégio Betânia",
		},
		{
			"História Geral de África — Metodologia e Pré-História (Volume I)",
			"UNESCO / Comité Internacional",
			"historia",
			"geral",
			"Obra monumental e de referência global sobre o berço da humanidade, civilizações ancestrais e fontes orais africanas.",
			2010,
			"link",
			"https://unesdoc.unesco.org/ark:/48223/pf0000190249",
			"Curadoria Colégio Betânia",
		},
		{
			"Introdução à Lógica e Filosofia",
			"Prof. Universitário Open Access",
			"filosofia",
			"segundo-ciclo",
			"Tratado de introdução ao pensamento crítico, silogismos, argumentação ética e epistemologia.",
			2022,
			"link",
			"https://milnepublishing.geneseo.edu/introphilosophy/",
			"Curadoria Colégio Betânia",
		},
		{
			"Lógica de Programação e Algoritmos com Python",
			"Comunidade de Educação Aberta",
			"informatica",
			"primeiro-ciclo",
			"Aprenda a pensar como um programador: variáveis, condicionais, loops e criação de pequenos programas didáticos.",
			2025,
			"link",
			"https://greenteapress.com/wp/think-python-2e/",
			"Curadoria Colégio Betânia",
		},
		{
			"O Cortiço",
			"Aluísio Azevedo",
			"portugues-literatura",
			"segundo-ciclo",
			"Obra-prima do naturalismo que explora as tensões sociais, influências do meio e transformações urbanas do século XIX.",
			1890,
			"link",
			"https://www.dominiopublico.gov.br/download/texto/bv000008.pdf",
			"Curadoria Colégio Betânia",
		},
	}

	for _, o := range obras {
		var matID, nivID sql.NullInt64
		_ = db.SqlDB.QueryRow("SELECT id FROM disciplinas WHERE slug = ?", o.materiaSlug).Scan(&matID)
		_ = db.SqlDB.QueryRow("SELECT id FROM niveis WHERE slug = ?", o.nivelSlug).Scan(&nivID)

		_, err := db.SqlDB.Exec(`
			INSERT INTO livros (
				titulo, autor, disciplina_id, nivel_id, sinopse,
				ano_publicacao, formato, link_externo, status, submetido_por
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'ativo', ?)
		`, o.titulo, o.autor, matID, nivID, o.sinopse, o.ano, o.formato, o.link, o.submetidoPor)
		if err != nil {
			return fmt.Errorf("seed livro %s: %w", o.titulo, err)
		}
	}

	return nil
}
