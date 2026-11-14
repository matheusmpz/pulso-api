package calculos

import "math"

// Casas decimais da resposta (seção 8).
const (
	CasasPercentual = 1
	CasasValor      = 2 // R$ e horas
	CasasDias       = 1
)

// Arredondar arredonda v para n casas decimais (meio para longe do zero).
func Arredondar(v float64, casas int) float64 {
	p := math.Pow(10, float64(casas))
	return math.Round(v*p) / p
}

// Ptr devolve um ponteiro para v (valores opcionais no JSON).
func Ptr(v float64) *float64 {
	return &v
}

// PtrArredondado devolve um ponteiro para v arredondado.
func PtrArredondado(v float64, casas int) *float64 {
	return Ptr(Arredondar(v, casas))
}

// Percentual devolve num ÷ den × 100 com 1 casa, ou nil se den for 0
// (razão sem denominador vira null; o front mostra "—").
func Percentual(num, den float64) *float64 {
	if den == 0 {
		return nil
	}
	return PtrArredondado(num/den*100, CasasPercentual)
}

// Variacao devolve (atual − anterior) ÷ anterior × 100 com 1 casa, ou nil
// se o anterior for 0 (seção 7.3).
func Variacao(atual, anterior float64) *float64 {
	if anterior == 0 {
		return nil
	}
	return PtrArredondado((atual-anterior)/anterior*100, CasasPercentual)
}
