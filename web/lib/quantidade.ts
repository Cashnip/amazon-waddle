// O teto de unidades por Item de Carrinho (§7.1). Nasceu na 3.6 como constante
// do web/, e continua: o Carrinho da Épica 4 trouxe o mesmo limiar ao Go
// (AZAMON_CARRINHO_UNIDADES_MAX), que é quem decide; este só limita o seletor.
// Os dois têm de mudar juntos (deferred-work.md, 7.3).
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
