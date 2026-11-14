package utils

import "net/http"

// Mensagens padrão de acesso a recursos (dashboards e cards).
const (
	MsgSemAcesso = "Você não tem acesso a este recurso"
)

// Acesso é o resultado da regra de permissão de um recurso.
type Acesso int

const (
	// AcessoPermitido: o recurso existe, está ativo e o perfil pode vê-lo.
	AcessoPermitido Acesso = iota
	// AcessoNaoEncontrado: o recurso não existe ou está inativo (404).
	AcessoNaoEncontrado
	// AcessoNegado: o recurso existe e está ativo, mas o perfil não pode vê-lo (403).
	AcessoNegado
)

// AvaliarAcesso aplica a regra de acesso a um dashboard ou card:
// inexistente ou inativo → 404; sem permissão → 403; senão, permitido.
// A existência vem antes da permissão: um recurso inativo dá 404 mesmo
// para quem não teria acesso a ele.
func AvaliarAcesso(existe, ativo, permitido bool) Acesso {
	switch {
	case !existe || !ativo:
		return AcessoNaoEncontrado
	case !permitido:
		return AcessoNegado
	default:
		return AcessoPermitido
	}
}

// StatusHTTP devolve o status HTTP correspondente ao resultado.
func (a Acesso) StatusHTTP() int {
	switch a {
	case AcessoNaoEncontrado:
		return http.StatusNotFound
	case AcessoNegado:
		return http.StatusForbidden
	default:
		return http.StatusOK
	}
}
