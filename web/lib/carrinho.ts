// O Carrinho do lado do navegador (4.3). Nenhuma regra mora aqui (AD-10): quem
// decide teto, Estoque e preço é o Go, e a tela mostra a resposta dele. O que
// há é a forma da resposta, o aviso entre a barra e a tela, e a aritmética de
// centavos inteiros que a edição otimista da quantidade precisa no intervalo
// entre o clique e a resposta.

export type LinhaDoCarrinho = {
  id: string;
  produto_id: string;
  quantidade: number;
  // Falso quando o Produto saiu da visibilidade: nome, imagem e preço vêm
  // vazios, e a linha só serve para ser removida.
  visivel: boolean;
  nome: string;
  imagem_url: string;
  preco_centavos: number;
};

export type Carrinho = {
  itens: LinhaDoCarrinho[];
  unidades: number;
  subtotal_centavos: number;
};

// O aviso entre quem altera o Carrinho e o contador da barra: um evento na
// própria janela, e não consulta em intervalo (UX-DR16). A barra e a tela são
// árvores separadas de componentes, e não há estado compartilhado entre elas.
export const CARRINHO_ALTERADO = "azamon:carrinho";

export function avisarCarrinhoAlterado() {
  window.dispatchEvent(new Event(CARRINHO_ALTERADO));
}

export function unidadesDe(itens: LinhaDoCarrinho[]): number {
  return itens.reduce((soma, i) => soma + i.quantidade, 0);
}

// O subtotal soma só as linhas visíveis, como o Go: a linha indisponível não
// tem preço a somar. O total exibido é a soma exata das parcelas exibidas.
export function parcelaDe(linha: LinhaDoCarrinho): number {
  return linha.visivel ? linha.preco_centavos * linha.quantidade : 0;
}

export function subtotalDe(itens: LinhaDoCarrinho[]): number {
  return itens.reduce((soma, i) => soma + parcelaDe(i), 0);
}

// O estado da edição otimista: a quantidade nova na linha, e unidades e
// subtotal refeitos. É estimativa do intervalo até a resposta — quando ela
// chega, a tela recarrega o Carrinho e o número que vale é o do servidor.
export function comQuantidade(carrinho: Carrinho, itemId: string, quantidade: number): Carrinho {
  const itens = carrinho.itens.map((i) => (i.id === itemId ? { ...i, quantidade } : i));
  return { itens, unidades: unidadesDe(itens), subtotal_centavos: subtotalDe(itens) };
}

// O texto do campo de quantidade: só inteiro sem sinal. O que não é isso
// devolve null, e a tela repõe o valor que estava — o teto e o Estoque são do Go.
export function quantidadeDoCampo(texto: string): number | null {
  const limpo = texto.trim();
  if (!/^\d+$/.test(limpo)) return null;
  const n = Number(limpo);
  return Number.isSafeInteger(n) ? n : null;
}
