package repositories

import (
	"context"
	"database/sql"

	"pulso-api/models"
	"pulso-api/utils"
)

// MetricasIARepository calcula os números de consumo de IA (só leitura).
// Usado pelos cards de IA e pelo proj-03.
type MetricasIARepository interface {
	// ConsumoAcumulado: tokens de T até o fim do dia ate, quota do escopo
	// e custo de T no período [de, ate].
	ConsumoAcumulado(ctx context.Context, f models.Filtros) (models.ResumoConsumoIA, error)
	// ConsumoNoTempo: tokens de T acumulados por dia, do primeiro consumo
	// até ate, e a quota do escopo (proj-15).
	ConsumoNoTempo(ctx context.Context, f models.Filtros) (models.ConsumoNoTempo, error)
	// TokensPeriodo: custo e tokens (entrada e saída) de T no período
	// (ia-01, ia-02). Use outros de/ate em f para o período anterior.
	TokensPeriodo(ctx context.Context, f models.Filtros) (models.ResumoTokensPeriodo, error)
	// CustoAtividadesConcluidas: atividades de A concluídas no período e o
	// custo de IA ligado a elas até ate (ia-04).
	CustoAtividadesConcluidas(ctx context.Context, f models.Filtros) (models.ResumoCustoAtividades, error)
	// ConsumoDiario: tokens de T por dia, de 6 dias antes de de até ate
	// (com zeros), o acumulado até ate e a quota (ia-05).
	ConsumoDiario(ctx context.Context, f models.Filtros) (models.ConsumoDiario, error)
	// CustoPorProjeto: custo de T no período por projeto, decrescente (ia-06).
	CustoPorProjeto(ctx context.Context, f models.Filtros) ([]models.GrupoCusto, error)
	// CustoPorSprint: custo de T no período por sprint da atividade, com
	// "Sem sprint" para consumo sem atividade ou de backlog (ia-06 com
	// escopo=projeto e id).
	CustoPorSprint(ctx context.Context, f models.Filtros) ([]models.GrupoCusto, error)
	// CustoPorModelo: custo e tokens de T no período por modelo (ia-07).
	CustoPorModelo(ctx context.Context, f models.Filtros) ([]models.GrupoCusto, error)
	// CustoPorEtapa: custo de T no período por etapa (ia-08).
	CustoPorEtapa(ctx context.Context, f models.Filtros) ([]models.GrupoCusto, error)
	// CustoTipoComplexidade: custo de T no período e atividades distintas de
	// A com consumo, por tipo e complexidade (ia-09).
	CustoTipoComplexidade(ctx context.Context, f models.Filtros) ([]models.CustoTipoComplexidade, error)
}

type metricasIARepository struct {
	db *sql.DB
}

// NewMetricasIARepository cria o repository de métricas de IA.
func NewMetricasIARepository(db *sql.DB) MetricasIARepository {
	return &metricasIARepository{db}
}

func (r *metricasIARepository) ConsumoAcumulado(ctx context.Context, f models.Filtros) (models.ResumoConsumoIA, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim := c.Arg(inicio), c.Arg(fimAte)
	query := "WITH " + CTEsEscopo(c, f) + `
	SELECT (SELECT COALESCE(SUM(t.tokens_entrada::bigint + t.tokens_saida), 0)::bigint
	        FROM t WHERE t.ocorrido_em < ` + pFim + `::timestamptz),
	       (SELECT valor FROM quota),
	       (SELECT COALESCE(SUM(t.custo), 0)::float8
	        FROM t WHERE t.ocorrido_em >= ` + pIni + `::timestamptz AND t.ocorrido_em < ` + pFim + `::timestamptz)`

	var res models.ResumoConsumoIA
	err := r.db.QueryRowContext(ctx, query, c.Args()...).Scan(&res.TokensAcumulados, &res.Quota, &res.CustoPeriodo)
	return res, err
}
