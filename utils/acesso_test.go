package utils

import (
	"net/http"
	"testing"
)

func TestAvaliarAcesso(t *testing.T) {
	casos := []struct {
		nome                     string
		existe, ativo, permitido bool
		esperado                 Acesso
		status                   int
	}{
		{"permitido", true, true, true, AcessoPermitido, http.StatusOK},
		{"sem permissão", true, true, false, AcessoNegado, http.StatusForbidden},
		{"inativo com permissão", true, false, true, AcessoNaoEncontrado, http.StatusNotFound},
		{"inativo sem permissão", true, false, false, AcessoNaoEncontrado, http.StatusNotFound},
		{"inexistente", false, false, false, AcessoNaoEncontrado, http.StatusNotFound},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			obtido := AvaliarAcesso(c.existe, c.ativo, c.permitido)
			if obtido != c.esperado {
				t.Errorf("acesso esperado %v, obtido %v", c.esperado, obtido)
			}
			if obtido.StatusHTTP() != c.status {
				t.Errorf("status esperado %d, obtido %d", c.status, obtido.StatusHTTP())
			}
		})
	}
}
