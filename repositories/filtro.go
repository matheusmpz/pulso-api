package repositories

import (
	"context"
	"database/sql"
	"time"

	"pulso-api/models"
	"pulso-api/utils"
)

// FiltroRepository lê as opções dos filtros e as sprints (só leitura).
type FiltroRepository interface {
	// Origens devolve os valores distintos de oportunidades.origem.
	Origens(ctx context.Context) ([]string, error)
	// Modelos devolve os valores distintos de consumo_tokens.modelo.
	Modelos(ctx context.Context) ([]string, error)
	// Projetos devolve todos os projetos, ordenados por nome.
	Projetos(ctx context.Context) ([]models.ProjetoOpcao, error)
	// Times devolve todos os times, ordenados por nome.
	Times(ctx context.Context) ([]models.TimeOpcao, error)
	// Sprints devolve todas as sprints com o nome do time, ordenadas por
	// time e início.
	Sprints(ctx context.Context) ([]models.Sprint, error)
	// BuscarSprint devolve uma sprint pelo id (sql.ErrNoRows se não existir).
	BuscarSprint(ctx context.Context, id int64) (models.Sprint, error)
}

type filtroRepository struct {
	db *sql.DB
}

// NewFiltroRepository cria o repository de filtros.
func NewFiltroRepository(db *sql.DB) FiltroRepository {
	return &filtroRepository{db}
}

func (r *filtroRepository) Origens(ctx context.Context) ([]string, error) {
	return r.listarTextos(ctx, `SELECT DISTINCT origem FROM oportunidades ORDER BY origem`)
}

func (r *filtroRepository) Modelos(ctx context.Context) ([]string, error) {
	return r.listarTextos(ctx, `SELECT DISTINCT modelo FROM consumo_tokens ORDER BY modelo`)
}

func (r *filtroRepository) listarTextos(ctx context.Context, query string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	valores := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		valores = append(valores, v)
	}
	return valores, rows.Err()
}

func (r *filtroRepository) Projetos(ctx context.Context) ([]models.ProjetoOpcao, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, nome, tipo::text, status::text
		FROM projetos
		ORDER BY nome, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projetos := []models.ProjetoOpcao{}
	for rows.Next() {
		var p models.ProjetoOpcao
		if err := rows.Scan(&p.ID, &p.Nome, &p.Tipo, &p.Status); err != nil {
			return nil, err
		}
		projetos = append(projetos, p)
	}
	return projetos, rows.Err()
}

func (r *filtroRepository) Times(ctx context.Context) ([]models.TimeOpcao, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, nome FROM times ORDER BY nome, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	times := []models.TimeOpcao{}
	for rows.Next() {
		var t models.TimeOpcao
		if err := rows.Scan(&t.ID, &t.Nome); err != nil {
			return nil, err
		}
		times = append(times, t)
	}
	return times, rows.Err()
}

const selectSprint = `
	SELECT s.id, s.time_id, t.nome, s.numero, s.inicio, s.fim
	FROM sprints s
	JOIN times t ON t.id = s.time_id`

func (r *filtroRepository) Sprints(ctx context.Context) ([]models.Sprint, error) {
	rows, err := r.db.QueryContext(ctx, selectSprint+` ORDER BY t.nome, s.inicio, s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sprints := []models.Sprint{}
	for rows.Next() {
		s, err := escanearSprint(rows)
		if err != nil {
			return nil, err
		}
		sprints = append(sprints, s)
	}
	return sprints, rows.Err()
}

func (r *filtroRepository) BuscarSprint(ctx context.Context, id int64) (models.Sprint, error) {
	return escanearSprint(r.db.QueryRowContext(ctx, selectSprint+` WHERE s.id = $1`, id))
}

// escaneavel é o que *sql.Row e *sql.Rows têm em comum.
type escaneavel interface {
	Scan(dest ...interface{}) error
}

func escanearSprint(row escaneavel) (models.Sprint, error) {
	var s models.Sprint
	var inicio, fim time.Time
	if err := row.Scan(&s.ID, &s.TimeID, &s.TimeNome, &s.Numero, &inicio, &fim); err != nil {
		return models.Sprint{}, err
	}
	// DATE chega como meia-noite UTC; mantém o mesmo dia no fuso do app
	s.Inicio, s.Fim = utils.DataDoBanco(inicio), utils.DataDoBanco(fim)
	return s, nil
}
