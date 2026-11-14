package models

// Resultados tipados dos repositories de métricas. Os Calculadores
// transformam esses números no formato de "dados" de cada card.

// ResumoProgresso alimenta o proj-01 (Evolução % ponderada pela estimativa).
type ResumoProgresso struct {
	EstimativaConcluida float64 // soma de estimativa_horas das atividades concluídas
	EstimativaTotal     float64 // soma de estimativa_horas de todas as atividades de A
	Concluidas          int
	Total               int
}

// ResumoHoras alimenta o proj-02 (horas reais contra estimadas).
type ResumoHoras struct {
	Reais     float64 // soma de horas de H até ate
	Estimadas float64 // soma de estimativa_horas de A
}

// ResumoConsumoIA alimenta o proj-03 (e depois o ia-03): consumo de
// tokens contra a quota do escopo.
type ResumoConsumoIA struct {
	TokensAcumulados int64   // tokens (entrada + saída) de T até ate
	Quota            int64   // soma de limite_tokens dos projetos de P
	CustoPeriodo     float64 // custo (R$) de T entre de e ate
}
