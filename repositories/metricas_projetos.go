package repositories

import (
	"context"
	"database/sql"

	"pulso-api/models"
	"pulso-api/utils"
)

// MetricasProjetosRepository calcula os números dos cards do dashboard
// projetos (só leitura). Cada tabela é agregada separadamente; nada de
// JOIN entre tabelas de evento para somar.
type MetricasProjetosRepository interface {
	// Progresso: estimativa das atividades de A concluídas até o fim do
	// dia ate, sobre a estimativa total de A (proj-01).
	Progresso(ctx context.Context, f models.Filtros) (models.ResumoProgresso, error)
	// Horas: horas de H lançadas até ate e estimativa total de A (proj-02).
	Horas(ctx context.Context, f models.Filtros) (models.ResumoHoras, error)
}

type metricasProjetosRepository struct {
	db *sql.DB
}

// NewMetricasProjetosRepository cria o repository de métricas de projetos.
func NewMetricasProjetosRepository(db *sql.DB) MetricasProjetosRepository {
	return &metricasProjetosRepository{db}
}

func (r *metricasProjetosRepository) Progresso(ctx context.Context, f models.Filtros) (models.ResumoProgresso, error) {
	c := &ConsultaSQL{}
	_, fimAte := utils.Intervalo(f.De, f.Ate)
	pFim := c.Arg(fimAte)
	query := "WITH " + CTEsEscopo(c, f) + `,
	situacao AS (
		SELECT a.estimativa_horas, ` + StatusAtividadeAte("a.id", pFim) + ` AS status
		FROM a
	)
	SELECT COALESCE(SUM(estimativa_horas) FILTER (WHERE status = 'concluida'), 0)::float8,
	       COALESCE(SUM(estimativa_horas), 0)::float8,
	       COUNT(*) FILTER (WHERE status = 'concluida'),
	       COUNT(*)
	FROM situacao`

	var p models.ResumoProgresso
	err := r.db.QueryRowContext(ctx, query, c.Args()...).Scan(&p.EstimativaConcluida, &p.EstimativaTotal, &p.Concluidas, &p.Total)
	return p, err
}

func (r *metricasProjetosRepository) Horas(ctx context.Context, f models.Filtros) (models.ResumoHoras, error) {
	c := &ConsultaSQL{}
	pAte := c.Arg(f.Ate.String())
	// Dois agregados independentes (h e a), cada um na sua subconsulta
	query := "WITH " + CTEsEscopo(c, f) + `
	SELECT (SELECT COALESCE(SUM(h.horas), 0)::float8 FROM h WHERE h.trabalhado_em <= ` + pAte + `::date),
	       (SELECT COALESCE(SUM(a.estimativa_horas), 0)::float8 FROM a)`

	var h models.ResumoHoras
	err := r.db.QueryRowContext(ctx, query, c.Args()...).Scan(&h.Reais, &h.Estimadas)
	return h, err
}
