package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"biblioteca-betania/internal/models"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// DB engloba o banco de dados com mutex de proteção de concorrência
type DB struct {
	SqlDB *sql.DB
	mu    sync.RWMutex
}

// Open inicializa a conexão com SQLite em modo WAL seguro
func Open(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("falha ao criar pasta do banco: %w", err)
	}

	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", dbPath)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir sqlite: %w", err)
	}

	// Conexões seguras: leituras concorrentes suportadas pelo WAL
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	db := &DB{SqlDB: sqlDB}
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("falha ao aplicar schema: %w", err)
	}

	if err := db.SeedInitialData(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("falha ao rodar seed inicial: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.SqlDB.Close()
}

func (db *DB) migrate() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.SqlDB.Exec(schemaSQL)
	return err
}

// -------------------------------------------------------------
// Operações de Disciplinas e Níveis
// -------------------------------------------------------------

func (db *DB) ObterDisciplinas() ([]models.Disciplina, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `SELECT id, nome, slug, icone, ordem FROM disciplinas ORDER BY ordem ASC, nome ASC`
	rows, err := db.SqlDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.Disciplina
	for rows.Next() {
		var d models.Disciplina
		if err := rows.Scan(&d.ID, &d.Nome, &d.Slug, &d.Icone, &d.Ordem); err != nil {
			return nil, err
		}
		lista = append(lista, d)
	}
	return lista, rows.Err()
}

func (db *DB) ObterNiveis() ([]models.NivelEducacional, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `SELECT id, nome, slug FROM niveis ORDER BY id ASC`
	rows, err := db.SqlDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.NivelEducacional
	for rows.Next() {
		var n models.NivelEducacional
		if err := rows.Scan(&n.ID, &n.Nome, &n.Slug); err != nil {
			return nil, err
		}
		lista = append(lista, n)
	}
	return lista, rows.Err()
}

// -------------------------------------------------------------
// Catálogo de Livros e FTS5 (Totalmente imune a SQL Injection)
// -------------------------------------------------------------

type FiltrosBusca struct {
	Query        string
	DisciplinaID int64
	NivelID      int64
	Pagina       int
	Limite       int
}

func (db *DB) ListarLivros(f FiltrosBusca) ([]models.Livro, int64, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if f.Pagina < 1 {
		f.Pagina = 1
	}
	if f.Limite < 1 || f.Limite > 100 {
		f.Limite = 24
	}
	offset := (f.Pagina - 1) * f.Limite

	var condicoes []string
	var args []any

	condicoes = append(condicoes, "l.status = 'ativo'")

	if f.DisciplinaID > 0 {
		condicoes = append(condicoes, "l.disciplina_id = ?")
		args = append(args, f.DisciplinaID)
	}
	if f.NivelID > 0 {
		condicoes = append(condicoes, "l.nivel_id = ?")
		args = append(args, f.NivelID)
	}

	whereFTS := ""
	if f.Query != "" {
		termoLimpo := sanitizarTermoFTS(f.Query)
		if termoLimpo != "" {
			// FTS5 MATCH seguro sanitizando termos para evitar erro de sintaxe
			whereFTS = " INNER JOIN livros_fts fts ON fts.rowid = l.id AND livros_fts MATCH ?"
			args = append([]any{termoLimpo}, args...)
		} else {
			// Se o termo consistir apenas de caracteres especiais/ataque, não retorna nada
			condicoes = append(condicoes, "1 = 0")
		}
	}

	whereClause := ""
	if len(condicoes) > 0 {
		whereClause = " WHERE " + joinStrings(condicoes, " AND ")
	}

	// 1. Contagem total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM livros l%s%s", whereFTS, whereClause)
	var total int64
	if err := db.SqlDB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 2. Busca paginada
	selectQuery := fmt.Sprintf(`
		SELECT 
			l.id, l.titulo, l.autor, l.disciplina_id, COALESCE(d.nome, ''),
			l.nivel_id, COALESCE(n.nome, ''), l.sinopse, l.ano_publicacao,
			l.formato, COALESCE(l.arquivo_url, ''), COALESCE(l.link_externo, ''),
			l.tamanho_bytes, l.status, l.submetido_por, l.visualizacoes, l.downloads, l.criado_em
		FROM livros l
		LEFT JOIN disciplinas d ON d.id = l.disciplina_id
		LEFT JOIN niveis n ON n.id = l.nivel_id
		%s
		%s
		ORDER BY l.id DESC
		LIMIT ? OFFSET ?
	`, whereFTS, whereClause)

	queryArgs := append(args, f.Limite, offset)
	rows, err := db.SqlDB.Query(selectQuery, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var livros []models.Livro
	for rows.Next() {
		var l models.Livro
		var discID, nivID sql.NullInt64
		if err := rows.Scan(
			&l.ID, &l.Titulo, &l.Autor, &discID, &l.DisciplinaNome,
			&nivID, &l.NivelNome, &l.Sinopse, &l.AnoPublicacao,
			&l.Formato, &l.ArquivoURL, &l.LinkExterno,
			&l.TamanhoBytes, &l.Status, &l.SubmetidoPor, &l.Visualizacoes, &l.Downloads, &l.CriadoEm,
		); err != nil {
			return nil, 0, err
		}
		if discID.Valid {
			l.DisciplinaID = discID.Int64
		}
		if nivID.Valid {
			l.NivelID = nivID.Int64
		}
		livros = append(livros, l)
	}

	return livros, total, rows.Err()
}

func (db *DB) ObterLivroPorID(id int64) (*models.Livro, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `
		SELECT 
			l.id, l.titulo, l.autor, l.disciplina_id, COALESCE(d.nome, ''),
			l.nivel_id, COALESCE(n.nome, ''), l.sinopse, l.ano_publicacao,
			l.formato, COALESCE(l.arquivo_url, ''), COALESCE(l.link_externo, ''),
			l.tamanho_bytes, l.status, l.submetido_por, l.visualizacoes, l.downloads, l.criado_em
		FROM livros l
		LEFT JOIN disciplinas d ON d.id = l.disciplina_id
		LEFT JOIN niveis n ON n.id = l.nivel_id
		WHERE l.id = ?
	`
	var l models.Livro
	var discID, nivID sql.NullInt64
	err := db.SqlDB.QueryRow(query, id).Scan(
		&l.ID, &l.Titulo, &l.Autor, &discID, &l.DisciplinaNome,
		&nivID, &l.NivelNome, &l.Sinopse, &l.AnoPublicacao,
		&l.Formato, &l.ArquivoURL, &l.LinkExterno,
		&l.TamanhoBytes, &l.Status, &l.SubmetidoPor, &l.Visualizacoes, &l.Downloads, &l.CriadoEm,
	)
	if discID.Valid {
		l.DisciplinaID = discID.Int64
	}
	if nivID.Valid {
		l.NivelID = nivID.Int64
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (db *DB) IncrementarVisualizacao(id int64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.SqlDB.Exec("UPDATE livros SET visualizacoes = visualizacoes + 1 WHERE id = ?", id)
	return err
}

func (db *DB) IncrementarDownload(id int64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.SqlDB.Exec("UPDATE livros SET downloads = downloads + 1 WHERE id = ?", id)
	return err
}

func (db *DB) InserirLivro(l *models.Livro) (int64, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	var discID, nivID any
	if l.DisciplinaID > 0 {
		discID = l.DisciplinaID
	}
	if l.NivelID > 0 {
		nivID = l.NivelID
	}

	query := `
		INSERT INTO livros (
			titulo, autor, disciplina_id, nivel_id, sinopse, ano_publicacao,
			formato, arquivo_url, link_externo, tamanho_bytes, status, submetido_por
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := db.SqlDB.Exec(query,
		l.Titulo, l.Autor, discID, nivID, l.Sinopse, l.AnoPublicacao,
		l.Formato, l.ArquivoURL, l.LinkExterno, l.TamanhoBytes, l.Status, l.SubmetidoPor,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// -------------------------------------------------------------
// Submissões de Estudantes & Fila da IA
// -------------------------------------------------------------

func (db *DB) InserirSubmissao(s *models.SubmissaoEstudante) (int64, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	var discID, nivID any
	if s.DisciplinaID > 0 {
		discID = s.DisciplinaID
	}
	if s.NivelID > 0 {
		nivID = s.NivelID
	}

	query := `
		INSERT INTO submissoes_estudantes (
			estudante_nome, estudante_classe, titulo, autor,
			disciplina_id, nivel_id, tipo_envio, arquivo_path,
			arquivo_nome_original, link_url, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pendente')
	`
	res, err := db.SqlDB.Exec(query,
		s.EstudanteNome, s.EstudanteClasse, s.Titulo, s.Autor,
		discID, nivID, s.TipoEnvio, s.ArquivoPath,
		s.ArquivoNomeOriginal, s.LinkURL,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) ObterSubmissoesPendentes(limite int) ([]models.SubmissaoEstudante, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `
		SELECT 
			id, estudante_nome, estudante_classe, titulo, COALESCE(autor, ''),
			COALESCE(disciplina_id, 0), COALESCE(nivel_id, 0), tipo_envio,
			COALESCE(arquivo_path, ''), COALESCE(arquivo_nome_original, ''),
			COALESCE(link_url, ''), status, COALESCE(ai_parecer, ''), criado_em
		FROM submissoes_estudantes
		WHERE status = 'pendente'
		ORDER BY id ASC
		LIMIT ?
	`
	rows, err := db.SqlDB.Query(query, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.SubmissaoEstudante
	for rows.Next() {
		var s models.SubmissaoEstudante
		if err := rows.Scan(
			&s.ID, &s.EstudanteNome, &s.EstudanteClasse, &s.Titulo, &s.Autor,
			&s.DisciplinaID, &s.NivelID, &s.TipoEnvio,
			&s.ArquivoPath, &s.ArquivoNomeOriginal,
			&s.LinkURL, &s.Status, &s.AIParecer, &s.CriadoEm,
		); err != nil {
			return nil, err
		}
		lista = append(lista, s)
	}
	return lista, rows.Err()
}

func (db *DB) AtualizarResultadoAvaliacaoIA(ctx context.Context, submissaoID int64, status, parecer string, livroID *int64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	query := `
		UPDATE submissoes_estudantes
		SET status = ?, ai_parecer = ?, ai_avaliado_em = CURRENT_TIMESTAMP, livro_id = ?
		WHERE id = ?
	`
	_, err := db.SqlDB.ExecContext(ctx, query, status, parecer, livroID, submissaoID)
	return err
}

func (db *DB) ListarSubmissoesRecentes(limite int) ([]models.SubmissaoEstudante, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `
		SELECT 
			id, estudante_nome, estudante_classe, titulo, COALESCE(autor, ''),
			COALESCE(disciplina_id, 0), COALESCE(nivel_id, 0), tipo_envio,
			COALESCE(arquivo_path, ''), COALESCE(arquivo_nome_original, ''),
			COALESCE(link_url, ''), status, COALESCE(ai_parecer, ''), criado_em
		FROM submissoes_estudantes
		ORDER BY id DESC
		LIMIT ?
	`
	rows, err := db.SqlDB.Query(query, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.SubmissaoEstudante
	for rows.Next() {
		var s models.SubmissaoEstudante
		if err := rows.Scan(
			&s.ID, &s.EstudanteNome, &s.EstudanteClasse, &s.Titulo, &s.Autor,
			&s.DisciplinaID, &s.NivelID, &s.TipoEnvio,
			&s.ArquivoPath, &s.ArquivoNomeOriginal,
			&s.LinkURL, &s.Status, &s.AIParecer, &s.CriadoEm,
		); err != nil {
			return nil, err
		}
		lista = append(lista, s)
	}
	return lista, rows.Err()
}

func (db *DB) ObterEstatisticas() (*models.EstatisticasAcervo, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var stats models.EstatisticasAcervo
	stats.MetaGlobal = 100000000 // 100 Milhões
	stats.LivrosPorMateria = make(map[string]int64)

	_ = db.SqlDB.QueryRow("SELECT COUNT(*) FROM livros WHERE status = 'ativo'").Scan(&stats.TotalLivros)
	_ = db.SqlDB.QueryRow("SELECT COUNT(*) FROM submissoes_estudantes").Scan(&stats.TotalSubmissoes)
	_ = db.SqlDB.QueryRow("SELECT COUNT(*) FROM submissoes_estudantes WHERE status = 'aprovado'").Scan(&stats.TotalAprovadosIA)

	rows, err := db.SqlDB.Query(`
		SELECT d.nome, COUNT(l.id)
		FROM disciplinas d
		LEFT JOIN livros l ON l.disciplina_id = d.id AND l.status = 'ativo'
		GROUP BY d.id
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var nome string
			var count int64
			if err := rows.Scan(&nome, &count); err == nil {
				stats.LivrosPorMateria[nome] = count
			}
		}
	}

	return &stats, nil
}

func joinStrings(elems []string, sep string) string {
	switch len(elems) {
	case 0:
		return ""
	case 1:
		return elems[0]
	}
	n := len(sep) * (len(elems) - 1)
	for i := 0; i < len(elems); i++ {
		n += len(elems[i])
	}

	b := make([]byte, 0, n)
	b = append(b, elems[0]...)
	for _, s := range elems[1:] {
		b = append(b, sep...)
		b = append(b, s...)
	}
	return string(b)
}

func sanitizarTermoFTS(q string) string {
	var b strings.Builder
	for _, r := range q {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	tokens := strings.Fields(b.String())
	if len(tokens) == 0 {
		return ""
	}
	var cleanTokens []string
	for _, t := range tokens {
		upper := strings.ToUpper(t)
		if upper == "OR" || upper == "AND" || upper == "NOT" {
			continue
		}
		cleanTokens = append(cleanTokens, "\""+t+"\"*")
	}
	if len(cleanTokens) == 0 {
		return ""
	}
	return strings.Join(cleanTokens, " ")
}
