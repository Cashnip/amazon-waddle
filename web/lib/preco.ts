// Reais ↔ centavos por texto, nunca por ponto flutuante (AD-3): "12,90" vira
// 1290 juntando os dígitos, e não multiplicando 12.9 por 100. O formato é o
// brasileiro — ponto só como separador de milhar, vírgula como decimal.
// Texto fora dele devolve null, e a tela pede o preço de novo; quem decide se
// o valor é aceitável (maior que zero, teto) é o Go.
export function reaisParaCentavos(texto: string): number | null {
  const limpo = texto.replace(/^\s*(R\$)?\s*/, "").trim();
  const m = /^(\d{1,3}(?:\.\d{3})+|\d+)(?:,(\d{1,2}))?$/.exec(limpo);
  if (!m) return null;
  const centavos = Number(m[1].replaceAll(".", "") + (m[2] ?? "").padEnd(2, "0"));
  return Number.isSafeInteger(centavos) ? centavos : null;
}

// O inverso, para preencher o formulário: 1290 vira "12,90".
export function centavosParaReais(centavos: number): string {
  return `${Math.floor(centavos / 100)},${String(centavos % 100).padStart(2, "0")}`;
}

// O preço como a tela o mostra: 129990 vira "R$ 1.299,90".
export function formatarPreco(centavos: number): string {
  const reais = Math.floor(centavos / 100).toLocaleString("pt-BR");
  return `R$ ${reais},${String(centavos % 100).padStart(2, "0")}`;
}
