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

// GrupoTempoMedio alimenta o proj-10 (por complexidade) e o proj-11 (por
// tipo de atividade): atividades de A concluídas no período.
type GrupoTempoMedio struct {
	Grupo         string  // valor do enum (ex.: "media", "correcao")
	N             int     // quantidade de atividades
	MediaReal     float64 // média das horas lançadas totais da atividade
	MediaEstimada float64 // média de estimativa_horas
}

// AtividadeEstourada alimenta o proj-12: atividade de A com horas reais
// acima da estimativa.
type AtividadeEstourada struct {
	Titulo     string
	Projeto    string
	Estimativa float64
	Reais      float64
}

// Estouros alimenta o proj-12: as maiores (top) e a quantidade total.
type Estouros struct {
	Top   []AtividadeEstourada
	Total int
}

// SprintHistorico alimenta o proj-13.
type SprintHistorico struct {
	Sprint    Sprint
	Planejado float64 // estimativa das atividades alocadas à sprint
	Concluido float64 // estimativa das concluídas ao fim de min(fim da sprint, ate)
	Destaque  bool    // a sprint do filtro (escopo=sprint)
}

// GrupoHoras alimenta o proj-14: horas por desenvolvedor ou por projeto.
type GrupoHoras struct {
	Rotulo string
	Horas  float64
}

// DiaConsumo alimenta o proj-15: tokens acumulados até o fim do dia.
type DiaConsumo struct {
	Dia              Data
	TokensAcumulados int64
}

// ConsumoNoTempo alimenta o proj-15.
type ConsumoNoTempo struct {
	Quota int64
	Dias  []DiaConsumo
}

// ResumoReceita alimenta o ven-01 e o ven-03: vendas fechadas num período.
type ResumoReceita struct {
	Valor  float64
	Vendas int
}

// ResumoConversao alimenta o ven-02: oportunidades encerradas no período.
type ResumoConversao struct {
	Ganhas, Perdidas int
}

// ResumoCiclo alimenta o ven-04: oportunidades ganhas no período.
type ResumoCiclo struct {
	MediaDias *float64 // nil se nenhuma
	Ganhas    int
}

// EtapaFunil alimenta o ven-05: oportunidades distintas com ao menos um
// evento da etapa no período.
type EtapaFunil struct {
	Etapa         string
	Oportunidades int
}

// EtapaPipeline alimenta o ven-06: valor em negociação por etapa atual.
type EtapaPipeline struct {
	Etapa     string
	Bruto     float64 // soma do valor do orçamento de maior versão
	Ponderado float64 // bruto × probabilidade da etapa
}

// GrupoContagem é um rótulo com uma contagem (ven-07 motivos de perda).
type GrupoContagem struct {
	Rotulo string
	N      int
}

// OrigemLeads alimenta o ven-08: leads criados no período por origem e
// o desfecho deles.
type OrigemLeads struct {
	Origem   string
	Leads    int
	Ganhas   int
	Perdidas int
}

// ReunioesDesfecho alimenta o ven-09: média de reuniões por oportunidade
// encerrada no período, por desfecho.
type ReunioesDesfecho struct {
	Desfecho string   // "ganha" ou "perdida"
	Media    *float64 // nil se não houver oportunidades
	N        int
}
