// O teto de unidades por Item de Carrinho (§7.1). Na 3.6 é constante do web/;
// ponytail: quando o Carrinho (Épica 4) nascer, o teto passa a vir do Go.
export const TETO_POR_ITEM = 10;

// A quantidade que o seletor aceita: 1 a min(10, disponível). É a mesma conta
// para o seletor e para o `?quantidade` que volta do Login.
export function quantidadeMaxima(disponivel: number): number {
  return Math.max(1, Math.min(TETO_POR_ITEM, disponivel));
}

// `?quantidade` vem da URL: acima do teto vira o teto, e o que não é inteiro
// positivo ("abc", "0", "-2", "1.5") vira 1.
export function quantidadeDaUrl(valor: string | string[] | undefined, disponivel: number): number {
  const texto = Array.isArray(valor) ? valor[0] : valor;
  if (!texto || !/^\d+$/.test(texto) || Number(texto) < 1) return 1;
  return Math.min(Number(texto), quantidadeMaxima(disponivel));
}
