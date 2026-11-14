package calculos

import "pulso-api/repositories"

// Constantes de código da seção 12 (não são variáveis de ambiente nem
// colunas). Só o ia-12 usa limiares; o campo "alerta" do envelope continua
// null em todos os cards.
const (
	LimiarQuotaAtencao = 80.0 // % da quota a partir do qual o projeto entra nos alertas
	LimiarQuotaCritica = 100.0
	FatorPico          = 3.0 // consumo do dia acima de 3× a média dos dias anteriores
	JanelaMediaDias    = repositories.JanelaMediaDias
	AmostraMinima      = 5 // ia-10: faixas com menos atividades são "amostra pequena"
)
