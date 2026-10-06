// Package expensas: conceptos, gastos, liquidaciones (máquina de estados),
// snapshots, distribución por mayor resto y cuenta corriente por UF.
//
// Dominios:
//   - conceptos: conceptos de expensa por consorcio (o de tenant cuando
//     consorcio_id es NULL), con categoría y regla 'coeficiente'.
//   - gastos: gastos asociados a un consorcio y su concepto de expensa.
//   - liquidaciones: liquidaciones por consorcio y período, con máquina de
//     estados (borrador → calculada/confirmada/publicada/cerrada, anulación)
//     y validación de fechas de vencimiento.
package expensas
