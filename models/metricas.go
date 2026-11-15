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

// MesReceita alimenta o ven-10: receita do mês e meta (nil se não houver).
type MesReceita struct {
	Mes     string // "AAAA-MM"
	Receita float64
	Meta    *float64
}

// StatusPropostas alimenta o ven-11: orçamentos emitidos no período por
// status atual.
type StatusPropostas struct {
	Pendentes, Aprovadas, Recusadas, EmRevisao int
}

// MargemComplexidade alimenta o ven-12.
type MargemComplexidade struct {
	Complexidade string
	N            int
	MargemMedia  float64 // média de margem_prevista (%)
	Valor        float64 // soma de valor
	Horas        float64 // soma de horas_estimadas
}

// OrcadoRealTipo alimenta o ven-13 e o ven-14: orçamentos vendidos no
// período, por tipo de projeto, e o realizado nos projetos ligados.
type OrcadoRealTipo struct {
	Tipo           string
	HorasEstimadas float64 // soma de orcamentos.horas_estimadas
	HorasReais     float64 // horas lançadas até ate nos projetos ligados
	MargemPrevista float64 // soma de valor × margem_prevista ÷ 100
	ValorVendido   float64 // soma de vendas.valor
	CustoHoras     float64 // soma de horas × custo_hora até ate
	CustoIA        float64 // soma do custo de IA até ate
}

// ResumoTokensPeriodo alimenta o ia-01 e o ia-02: consumo de T no período.
type ResumoTokensPeriodo struct {
	Custo   float64
	Entrada int64
	Saida   int64
}

// ResumoCustoAtividades alimenta o ia-04: atividades de A concluídas no
// período e o custo de IA ligado a elas.
type ResumoCustoAtividades struct {
	Atividades int
	Custo      float64
}

// DiaTokens é o consumo de tokens de um dia (ia-05).
type DiaTokens struct {
	Dia    Data
	Tokens int64
}

// ConsumoDiario alimenta o ia-05: tokens por dia desde 6 dias antes de de
// até ate (com zeros), o acumulado até ate e a quota.
type ConsumoDiario struct {
	Dias             []DiaTokens
	TokensAcumulados int64
	Quota            int64
}

// GrupoCusto é um rótulo com custo de IA e tokens (ia-06, ia-07, ia-08).
type GrupoCusto struct {
	Chave  string // valor bruto (ex.: modelo, etapa); vazio quando não se aplica
	Rotulo string
	Custo  float64
	Tokens int64
}

// CustoTipoComplexidade alimenta o ia-09: custo de IA no período e
// quantidade de atividades distintas por tipo e complexidade.
type CustoTipoComplexidade struct {
	Tipo         string
	Complexidade string
	Custo        float64
	Atividades   int
}

// FaixaIntensidade alimenta o ia-10: desvio médio das atividades de uma
// complexidade numa faixa (tercil) de intensidade de IA.
type FaixaIntensidade struct {
	Complexidade string
	Faixa        int     // 1 = baixa, 2 = média, 3 = alta
	DesvioMedio  float64 // média de horas reais ÷ estimativa × 100
	N            int
}

// PesoIAProjeto alimenta o ia-11: custo das horas e custo de IA no
// período, por projeto.
type PesoIAProjeto struct {
	Projeto    string
	CustoHoras float64
	CustoIA    float64
}

// AlertaQuota alimenta o ia-12: projeto com quota usada acima do limiar.
type AlertaQuota struct {
	Projeto    string
	Percentual float64
	Quando     Data // primeiro dia em que o acumulado passou do limiar de atenção
}

// AlertaPico alimenta o ia-12: dia com consumo acima de FatorPico × a
// média dos dias anteriores.
type AlertaPico struct {
	Projeto string
	Dia     Data
	Razao   float64 // consumo do dia ÷ média dos dias anteriores
}
