package utils

import (
	"reflect"
	"testing"
)

func TestVariaveisFaltando(t *testing.T) {
	casos := []struct {
		nome     string
		ambiente map[string]string
		esperado []string
	}{
		{
			nome:     "todas definidas",
			ambiente: map[string]string{"A": "1", "B": "2"},
			esperado: nil,
		},
		{
			nome:     "uma ausente",
			ambiente: map[string]string{"A": "1"},
			esperado: []string{"B"},
		},
		{
			nome:     "valor só com espaços conta como ausente",
			ambiente: map[string]string{"A": "   ", "B": "2"},
			esperado: []string{"A"},
		},
		{
			nome:     "todas ausentes mantêm a ordem",
			ambiente: map[string]string{},
			esperado: []string{"A", "B"},
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			buscar := func(n string) string { return c.ambiente[n] }
			obtido := VariaveisFaltando([]string{"A", "B"}, buscar)
			if !reflect.DeepEqual(obtido, c.esperado) {
				t.Errorf("esperado %v, obtido %v", c.esperado, obtido)
			}
		})
	}
}

func TestEnvOuPadrao(t *testing.T) {
	t.Setenv("PULSO_TESTE_DEFINIDA", "valor")
	t.Setenv("PULSO_TESTE_VAZIA", "")

	if v := EnvOuPadrao("PULSO_TESTE_DEFINIDA", "padrao"); v != "valor" {
		t.Errorf("esperado valor, obtido %q", v)
	}
	if v := EnvOuPadrao("PULSO_TESTE_VAZIA", "padrao"); v != "padrao" {
		t.Errorf("esperado padrao, obtido %q", v)
	}
}
