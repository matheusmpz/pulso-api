package utils

import (
	"fmt"
	"sync"
	"time"

	"pulso-api/models"
)

// FusoPadrao é usado quando APP_TIMEZONE não é informado.
const FusoPadrao = "America/Sao_Paulo"

var (
	fusoMu  sync.RWMutex
	fusoApp *time.Location
)

// ConfigurarFuso carrega o fuso da aplicação (APP_TIMEZONE). Deve ser
// chamado na inicialização; devolve erro se o nome não for um fuso válido.
func ConfigurarFuso(nome string) error {
	loc, err := time.LoadLocation(nome)
	if err != nil {
		return fmt.Errorf("APP_TIMEZONE inválido (%q): %w", nome, err)
	}
	fusoMu.Lock()
	fusoApp = loc
	fusoMu.Unlock()
	return nil
}

// FusoApp devolve o fuso da aplicação. Se ConfigurarFuso não foi chamado,
// tenta o padrão e, em último caso, UTC.
func FusoApp() *time.Location {
	fusoMu.RLock()
	loc := fusoApp
	fusoMu.RUnlock()
	if loc != nil {
		return loc
	}
	if padrao, err := time.LoadLocation(FusoPadrao); err == nil {
		return padrao
	}
	return time.UTC
}

// DataCivil converte um instante no dia do calendário correspondente,
// no fuso da aplicação (meia-noite).
func DataCivil(t time.Time) models.Data {
	t = t.In(FusoApp())
	return models.Data{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, FusoApp())}
}

// DataDoBanco converte uma coluna DATE lida pelo lib/pq (meia-noite em
// UTC) no mesmo dia do calendário no fuso da aplicação, sem mudar o dia.
func DataDoBanco(t time.Time) models.Data {
	return models.Data{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, FusoApp())}
}

// Hoje devolve o dia atual no fuso da aplicação.
func Hoje() models.Data {
	return DataCivil(time.Now())
}

// ParseData lê "AAAA-MM-DD" como um dia no fuso da aplicação.
func ParseData(valor string) (models.Data, error) {
	t, err := time.ParseInLocation(models.FormatoData, valor, FusoApp())
	if err != nil {
		return models.Data{}, err
	}
	return models.Data{Time: t}, nil
}

// SomarDias devolve a data deslocada em n dias (n pode ser negativo).
func SomarDias(d models.Data, n int) models.Data {
	return models.Data{Time: d.AddDate(0, 0, n)}
}

// DiasEntre devolve quantos dias vão de a até b (b − a), contando por
// dias do calendário.
func DiasEntre(a, b models.Data) int {
	ua := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	ub := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return int(ub.Sub(ua).Hours() / 24)
}

// PeriodoAnterior devolve o intervalo de mesma duração que termina no dia
// anterior a de (seção 7.3). Ex.: 11/11 a 20/11 (10 dias) → 01/11 a 10/11.
func PeriodoAnterior(de, ate models.Data) (models.Data, models.Data) {
	duracao := DiasEntre(de, ate) + 1
	ateAnterior := SomarDias(de, -1)
	return SomarDias(ateAnterior, -(duracao - 1)), ateAnterior
}

// Intervalo devolve os limites do período em instantes, para usar em SQL
// com TIMESTAMPTZ: [inicio do dia de, inicio do dia seguinte a ate).
func Intervalo(de, ate models.Data) (inicio, fimExclusivo time.Time) {
	return de.Time, SomarDias(ate, 1).Time
}
