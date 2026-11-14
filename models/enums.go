package models

// OpcaoEnum é um valor de enum do banco com o rótulo em português.
type OpcaoEnum struct {
	Valor  string `json:"valor" example:"web"`
	Rotulo string `json:"rotulo" example:"Web"`
}

// Valores dos enums do schema, na ordem de declaração, com os rótulos de
// exibição. Fonte: db/schema_v2.sql.
var (
	TiposProjeto = []OpcaoEnum{
		{"web", "Web"}, {"mobile", "Mobile"}, {"integracao", "Integração"},
		{"api", "API"}, {"outro", "Outro"},
	}
	StatusProjeto = []OpcaoEnum{
		{"em_andamento", "Em andamento"}, {"concluido", "Concluído"},
		{"pausado", "Pausado"}, {"cancelado", "Cancelado"},
	}
	EtapasTrabalho = []OpcaoEnum{
		{"analise", "Análise"}, {"desenvolvimento", "Desenvolvimento"}, {"testes", "Testes"},
	}
	Complexidades = []OpcaoEnum{
		{"pequena", "Pequena"}, {"media", "Média"}, {"grande", "Grande"},
	}
	TiposAtividade = []OpcaoEnum{
		{"analise", "Análise"}, {"nova_funcionalidade", "Nova funcionalidade"},
		{"melhoria", "Melhoria"}, {"correcao", "Correção"}, {"outro", "Outro"},
	}
	StatusAtividade = []OpcaoEnum{
		{"aberta", "Aberta"}, {"em_andamento", "Em andamento"}, {"concluida", "Concluída"},
	}
	EtapasFunil = []OpcaoEnum{
		{"lead", "Lead"}, {"qualificacao", "Qualificação"}, {"reuniao", "Reunião"},
		{"proposta_enviada", "Proposta enviada"}, {"negociacao", "Negociação"},
		{"ganha", "Ganha"}, {"perdida", "Perdida"},
	}
	StatusOrcamento = []OpcaoEnum{
		{"pendente", "Pendente"}, {"aprovada", "Aprovada"},
		{"recusada", "Recusada"}, {"em_revisao", "Em revisão"},
	}
)

// ValoresEnum devolve só os valores de uma lista de opções.
func ValoresEnum(opcoes []OpcaoEnum) []string {
	valores := make([]string, len(opcoes))
	for i, o := range opcoes {
		valores[i] = o.Valor
	}
	return valores
}

// RotuloEnum devolve o rótulo de um valor ou o próprio valor, se não houver.
func RotuloEnum(opcoes []OpcaoEnum, valor string) string {
	for _, o := range opcoes {
		if o.Valor == valor {
			return o.Rotulo
		}
	}
	return valor
}
