// A busca global da barra (3.10) é uma consulta nova: só termo e Categoria
// vão para a URL, e preço, ordenação e página ficam para trás.
export function destinoDaBusca(termo: string, categoria: string): string {
  const q = new URLSearchParams();
  if (termo.trim()) q.set("termo", termo.trim());
  if (categoria) q.set("categoria", categoria);
  const texto = q.toString();
  return texto ? `/?${texto}` : "/";
}

// O link de cada Categoria da Faixa.
export function destinoDaCategoria(id: string): string {
  return `/?${new URLSearchParams({ categoria: id })}`;
}
