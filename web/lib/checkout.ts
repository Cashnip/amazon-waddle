// A escolha de Endereço do checkout (5.2). O navegador guarda só o `id` do
// Endereço — nunca Frete, CEP ou total. É a pré-condição para trocar o
// Endereço recalcular o Frete: sem cópia no navegador, a Revisão terá de
// perguntar ao Go. O Frete em si é da 5.3 e da 5.4.
//
// Fica em `sessionStorage`, e não na URL, porque tem de atravessar a ida ao
// Carrinho e a volta (FR-22), e o botão do Carrinho não sabe o `id`. Toda
// leitura e escrita está em `try/catch`: janela privada, armazenamento
// bloqueado ou cota cheia equivalem a nada guardado.

export const CHAVE_DO_ENDERECO = "azamon:checkout:endereco_id";

type Armazenamento = Pick<Storage, "getItem" | "setItem">;

// O armazenamento é obtido por função, e dentro do `try`: em alguns navegadores
// é o próprio acesso a `sessionStorage` que lança. O parâmetro existe para o
// teste montar um armazenamento sem navegador.
const daSessao = (): Armazenamento => window.sessionStorage;

export function lerEscolha(armazenamento: () => Armazenamento = daSessao): string | null {
  try {
    const id = armazenamento().getItem(CHAVE_DO_ENDERECO);
    return id ? id : null;
  } catch {
    return null;
  }
}

// Devolve se guardou. Quem chama para no `false` e avisa o Comprador, em vez
// de seguir para uma Revisão que o mandaria de volta ao passo Endereço sem
// explicação.
export function guardarEscolha(id: string, armazenamento: () => Armazenamento = daSessao): boolean {
  try {
    armazenamento().setItem(CHAVE_DO_ENDERECO, id);
    return true;
  } catch {
    return false;
  }
}

// Qual Endereço vem marcado no passo Endereço: o `preferido` (o recém-
// cadastrado), se está na lista; senão o guardado, se ainda está na lista;
// senão o primeiro; senão nenhum. O guardado pode ter sido removido em "Meus
// endereços" entre uma visita e outra.
export function enderecoMarcado(
  lista: readonly { id: string }[],
  guardado: string | null,
  preferido: string | null = null,
): string | null {
  for (const candidato of [preferido, guardado]) {
    if (candidato !== null && lista.some((e) => e.id === candidato)) return candidato;
  }
  return lista.length > 0 ? lista[0].id : null;
}

// O Endereço que a Revisão mostra: só o guardado, e só se ainda está na lista.
// Aqui não há "o primeiro" — sem escolha válida, a Revisão volta ao passo
// Endereço, porque o Comprador não escolheu nada.
export function enderecoEscolhido<T extends { id: string }>(lista: readonly T[], guardado: string | null): T | null {
  if (guardado === null) return null;
  return lista.find((e) => e.id === guardado) ?? null;
}
