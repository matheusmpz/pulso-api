package repositories

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"pulso-api/models"
)

// EventoRepository lê os cursores das tabelas monitoradas pelo tempo real.
type EventoRepository interface {
	// MaxIDs devolve, para cada tabela monitorada (models.TabelaPorEvento),
	// o maior id atual (0 se vazia), numa única consulta.
	MaxIDs(ctx context.Context) (map[string]int64, error)
}

type eventoRepository struct {
	db *sql.DB
}

// NewEventoRepository cria o repository de eventos.
func NewEventoRepository(db *sql.DB) EventoRepository {
	return &eventoRepository{db}
}

// consultaMaxIDs é montada uma vez a partir da lista fixa de tabelas
// (whitelist em models.TabelaPorEvento; nada vem do usuário).
var consultaMaxIDs = func() string {
	var tabelas []string
	for _, t := range models.TabelaPorEvento {
		tabelas = append(tabelas, t)
	}
	sort.Strings(tabelas)
	partes := make([]string, len(tabelas))
	for i, t := range tabelas {
		partes[i] = "SELECT '" + t + "', COALESCE(MAX(id), 0) FROM " + t
	}
	return strings.Join(partes, " UNION ALL ")
}()

func (r *eventoRepository) MaxIDs(ctx context.Context) (map[string]int64, error) {
	rows, err := r.db.QueryContext(ctx, consultaMaxIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := map[string]int64{}
	for rows.Next() {
		var tabela string
		var id int64
		if err := rows.Scan(&tabela, &id); err != nil {
			return nil, err
		}
		ids[tabela] = id
	}
	return ids, rows.Err()
}
