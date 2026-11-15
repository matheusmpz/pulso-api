package calculos

import (
	"context"

	"pulso-api/models"
	"pulso-api/repositories"
)

// ---------------------------------------------------------------------
// proj-07 · Horas em análise, proj-08 · Horas em desenvolvimento,
// proj-09 · Horas em testes · barras_agrupadas · Acumulado
//
// Um único componente parametrizado pela etapa. Séries "estimado" (soma
// da estimativa da etapa nas atividades de A) e "real" (horas de H até
// ate lançadas na etapa). Um item:
//   {rotulo: "Análise" | "Desenvolvimento" | "Testes", estimado, real, percentual}
// percentual = real ÷ estimado × 100 (null se estimado for 0).
// ---------------------------------------------------------------------

type horasEtapa struct {
	codigo string
	etapa  string
	repo   repositories.MetricasProjetosRepository
}

// NovoHorasEtapa cria o calculador de horas de uma etapa
// (proj-07 análise, proj-08 desenvolvimento, proj-09 testes).
func NovoHorasEtapa(codigo, etapa string, repo repositories.MetricasProjetosRepository) Calculador {
	return horasEtapa{codigo: codigo, etapa: etapa, repo: repo}
}

func (c horasEtapa) Codigo() string     { return c.codigo }
func (c horasEtapa) Gatilhos() []string { return gatilhosPorCard[c.codigo] }

func (c horasEtapa) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	h, err := c.repo.HorasEtapa(ctx, f, c.etapa)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: MontarHorasEtapa(c.etapa, h)}, nil
}

// MontarHorasEtapa monta o barras_agrupadas de uma etapa.
func MontarHorasEtapa(etapa string, h models.HorasEtapa) models.DadosBarrasAgrupadas {
	var percentual interface{}
	if p := Percentual(h.Real, h.Estimado); p != nil {
		percentual = *p
	}
	return models.DadosBarrasAgrupadas{
		Series: []models.Serie{
			{Chave: "estimado", Nome: "Estimado"},
			{Chave: "real", Nome: "Real"},
		},
		Itens: []map[string]interface{}{{
			"rotulo":     models.RotuloEnum(models.EtapasTrabalho, etapa),
			"estimado":   Arredondar(h.Estimado, CasasValor),
			"real":       Arredondar(h.Real, CasasValor),
			"percentual": percentual,
		}},
		Unidade: "h",
	}
}
