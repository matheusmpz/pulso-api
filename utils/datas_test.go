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

func TestPeriodoAnterior(t *testing.T) {
	_ = ConfigurarFuso("America/Sao_Paulo")
	casos := []struct{ de, ate, deAnt, ateAnt string }{
		{"2026-11-11", "2026-11-20", "2026-11-01", "2026-11-10"}, // 10 dias
		{"2026-11-14", "2026-11-14", "2026-11-13", "2026-11-13"}, // 1 dia
		{"2026-03-01", "2026-03-31", "2026-01-29", "2026-02-28"}, // atravessa fevereiro
		{"2025-01-01", "2025-12-31", "2024-01-02", "2024-12-31"}, // 365 dias; 2024 é bissexto, então começa em 02/01
	}
	for _, c := range casos {
		de, _ := ParseData(c.de)
		ate, _ := ParseData(c.ate)
		gotDe, gotAte := PeriodoAnterior(de, ate)
		if gotDe.String() != c.deAnt || gotAte.String() != c.ateAnt {
			t.Errorf("%s a %s: esperado %s a %s, obtido %s a %s", c.de, c.ate, c.deAnt, c.ateAnt, gotDe, gotAte)
		}
	}
}

func TestDatasCivis(t *testing.T) {
	_ = ConfigurarFuso("America/Sao_Paulo")

	// 02:30 UTC de 15/11 ainda é 14/11 em São Paulo
	if d := DataCivil(time.Date(2026, 11, 15, 2, 30, 0, 0, time.UTC)); d.String() != "2026-11-14" {
		t.Errorf("DataCivil: esperado 2026-11-14, obtido %s", d)
	}
	// DATE do banco chega como meia-noite UTC e não pode mudar de dia
	if d := DataDoBanco(time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)); d.String() != "2026-11-03" {
		t.Errorf("DataDoBanco: esperado 2026-11-03, obtido %s", d)
	}

	de, _ := ParseData("2026-11-01")
	ate, _ := ParseData("2026-11-14")
	if DiasEntre(de, ate) != 13 {
		t.Errorf("DiasEntre: esperado 13, obtido %d", DiasEntre(de, ate))
	}
	inicio, fim := Intervalo(de, ate)
	if inicio.Format(time.RFC3339) != "2026-11-01T00:00:00-03:00" || fim.Format(time.RFC3339) != "2026-11-15T00:00:00-03:00" {
		t.Errorf("Intervalo inesperado: %s a %s", inicio.Format(time.RFC3339), fim.Format(time.RFC3339))
	}

	b, _ := de.MarshalJSON()
	if string(b) != `"2026-11-01"` {
		t.Errorf("JSON da data: esperado \"2026-11-01\", obtido %s", b)
	}
}
