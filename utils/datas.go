package utils

import (
	"fmt"
	"sync"
	"time"
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
