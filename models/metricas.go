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

// DiaBurnUp é um dia do burn-up (proj-04): horas de H acumuladas até o
// fim do dia e estimativa das atividades de A concluídas até o fim do dia.
type DiaBurnUp struct {
	Dia                 Data
	HorasAcumuladas     float64
	EstimativaConcluida float64
}

// BurnUp alimenta o proj-04.
type BurnUp struct {
	EstimativaTotal float64 // soma de estimativa_horas de A
	Dias            []DiaBurnUp
}

// ContagemStatus alimenta o proj-05: atividades de A por status atual.
type ContagemStatus struct {
	Abertas, EmAndamento, Concluidas int
}

// DiaBurndown é um dia da sprint no burndown (proj-06): estimativa das
// atividades da sprint ainda não concluídas ao fim do dia.
type DiaBurndown struct {
	Dia      Data
	Restante float64
}

// Burndown alimenta o proj-06. Sprint nil = nenhuma sprint alvo.
type Burndown struct {
	Sprint *Sprint
	Total  float64 // soma das estimativas das atividades da sprint em A
	Dias   []DiaBurndown
}

// HorasEtapa alimenta o proj-07/08/09: estimativa da etapa nas
// atividades de A e horas de H lançadas na etapa até ate.
type HorasEtapa struct {
	Estimado float64
	Real     float64
}
