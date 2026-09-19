// O Carrinho do lado do navegador (4.3). Nenhuma regra mora aqui (AD-10): quem
// decide teto, Estoque e preço é o Go, e a tela mostra a resposta dele. O que
// há é a forma da resposta, o aviso entre a barra e a tela, e a aritmética de
// centavos inteiros que a edição otimista da quantidade precisa no intervalo
// entre o clique e a resposta.

// O que impede a linha de seguir para o Pedido (FR-19). Quem decide é o Go, e a
// tela só mostra: vazio, `indisponivel` (só remover resolve) ou
// `acima_do_estoque` (remover, ou ajustar para o disponível).
export type Bloqueio = "" | "indisponivel" | "acima_do_estoque";

export type LinhaDoCarrinho = {
  id: string;
  produto_id: string;
  quantidade: number;
  // Falso quando o Produto saiu da visibilidade: nome, imagem, preço e Estoque
  // vêm vazios, e a linha só serve para ser removida.
  visivel: boolean;
  nome: string;
  imagem_url: string;
  preco_centavos: number;
  // O preço da última alteração da linha: o "de" do "de X para Y".
  preco_visto_centavos: number;
  estoque_disponivel: number;
  preco_mudou: boolean;
  bloqueio: Bloqueio;
};

export type Carrinho = {
  itens: LinhaDoCarrinho[];
  unidades: number;
  subtotal_centavos: number;
  // O limiar de isenção de Frete, da configuração do Go (AD-13). A tela mede a
  // distância até ele; nenhum valor de Frete existe aqui.
  frete_isencao_centavos: number;
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

// Quanto falta, em centavos, para o Frete grátis — e 0 quando o subtotal já
// alcançou o limiar, caso em que a tela não diz nada. Sai da mesma soma que o
// subtotal exibido, inclusive durante a edição otimista, porque a frase e o
// número ao lado dela não podem discordar (NFR-13, UX-DR17).
export function faltaParaFreteGratis(subtotal: number, isencao: number): number {
  return Math.max(0, isencao - subtotal);
}

// O estado da edição otimista: a quantidade nova na linha, e unidades e
// subtotal refeitos. É estimativa do intervalo até a resposta — quando ela
// chega, a tela recarrega o Carrinho e o número que vale é o do servidor.
export function comQuantidade(carrinho: Carrinho, itemId: string, quantidade: number): Carrinho {
  const itens = carrinho.itens.map((i) => (i.id === itemId ? { ...i, quantidade } : i));
  return { ...carrinho, itens, unidades: unidadesDe(itens), subtotal_centavos: subtotalDe(itens) };
}

// O texto do campo de quantidade: só inteiro sem sinal. O que não é isso
// devolve null, e a tela repõe o valor que estava — o teto e o Estoque são do Go.
export function quantidadeDoCampo(texto: string): number | null {
  const limpo = texto.trim();
  if (!/^\d+$/.test(limpo)) return null;
  const n = Number(limpo);
  return Number.isSafeInteger(n) ? n : null;
}

// A revalidação da FR-19 na abertura do Carrinho (4.4). Nada aqui vai ao
// servidor: confirmar um preço é estado da tela, e quem grava a ciência é o
// checkout (AD-17, 5.5) — por isso o aviso volta quando a tela é reaberta.

// O preço, em centavos, que o Comprador confirmou em cada linha. Se o preço
// mudar de novo, o valor guardado deixa de ser o atual e o aviso volta.
export type PrecosConfirmados = Record<string, number>;

export type Revalidacao = {
  // As linhas que impedem o Pedido, na ordem do Carrinho.
  bloqueios: LinhaDoCarrinho[];
  // As linhas de preço mudado que o Comprador ainda não confirmou.
  mudancas: LinhaDoCarrinho[];
  // Sem bloqueio e sem mudança pendente. Confirmar preço não desbloqueia: as
  // duas listas são independentes. A 5.1 lê isto quando o checkout existir.
  podeAvancar: boolean;
};

export function revalidacaoDe(carrinho: Carrinho, confirmados: PrecosConfirmados): Revalidacao {
  const bloqueios = carrinho.itens.filter((i) => i.bloqueio !== "");
  const mudancas = carrinho.itens.filter(
    (i) => i.visivel && i.preco_mudou && confirmados[i.id] !== i.preco_centavos,
  );
  return { bloqueios, mudancas, podeAvancar: bloqueios.length === 0 && mudancas.length === 0 };
}

export function comPrecosConfirmados(confirmados: PrecosConfirmados, linhas: LinhaDoCarrinho[]): PrecosConfirmados {
  const proximos = { ...confirmados };
  for (const l of linhas) proximos[l.id] = l.preco_centavos;
  return proximos;
}

// "1 unidade" e "2 unidades".
export function unidadesTexto(n: number): string {
  return `${n} ${n === 1 ? "unidade" : "unidades"}`;
}

// A frase do bloqueio. O Produto invisível não tem nome — desativado e de
// Vendedor desativado são o mesmo (FR-12) —, e o visível sem Estoque não diz
// "restam 0". O número é o Estoque disponível, e nunca o total (UX-DR17).
export function textoDoBloqueio(linha: LinhaDoCarrinho): string {
  if (!linha.visivel) return "Produto indisponível.";
  if (linha.bloqueio === "acima_do_estoque") {
    const d = linha.estoque_disponivel;
    return `${d === 1 ? "Resta" : "Restam"} ${unidadesTexto(d)} de ${linha.nome}; o Carrinho pede ${linha.quantidade}.`;
  }
  return `${linha.nome} está indisponível.`;
}
