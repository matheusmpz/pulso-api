package utils

import (
	"fmt"
	"os"
	"strings"
)

// VariaveisObrigatorias são as variáveis sem as quais a API não sobe.
var VariaveisObrigatorias = []string{"DB_HOST", "DB_USER", "DB_PASSWORD", "DB_NAME", "JWT_SECRET"}

// VariaveisFaltando devolve, na ordem recebida, os nomes cujo valor
// (lido por buscar) está vazio ou só tem espaços.
func VariaveisFaltando(nomes []string, buscar func(string) string) []string {
	var faltando []string
	for _, nome := range nomes {
		if strings.TrimSpace(buscar(nome)) == "" {
			faltando = append(faltando, nome)
		}
	}
	return faltando
}

// ValidarAmbiente confere as variáveis obrigatórias no ambiente do processo
// e devolve um erro com a lista das que faltam.
func ValidarAmbiente() error {
	if faltando := VariaveisFaltando(VariaveisObrigatorias, os.Getenv); len(faltando) > 0 {
		return fmt.Errorf("variáveis de ambiente obrigatórias não definidas: %s", strings.Join(faltando, ", "))
	}
	return nil
}

// EnvOuPadrao devolve o valor da variável ou o padrão, se ela estiver vazia.
func EnvOuPadrao(nome, padrao string) string {
	if v := strings.TrimSpace(os.Getenv(nome)); v != "" {
		return v
	}
	return padrao
}
