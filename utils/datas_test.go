package utils

import (
	"testing"
	"time"
	_ "time/tzdata" // base de fusos embutida, para o teste não depender do sistema
)

func TestConfigurarFuso(t *testing.T) {
	if err := ConfigurarFuso("Fuso/Inexistente"); err == nil {
		t.Error("fuso inválido deveria dar erro")
	}

	if err := ConfigurarFuso("America/Sao_Paulo"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if FusoApp().String() != "America/Sao_Paulo" {
		t.Errorf("fuso esperado America/Sao_Paulo, obtido %s", FusoApp())
	}

	// 2026-11-14 15:00 UTC = 12:00 em São Paulo (UTC-3, sem horário de verão)
	instante := time.Date(2026, 11, 14, 15, 0, 0, 0, time.UTC).In(FusoApp())
	if got := instante.Format(time.RFC3339); got != "2026-11-14T12:00:00-03:00" {
		t.Errorf("esperado 2026-11-14T12:00:00-03:00, obtido %s", got)
	}
}
