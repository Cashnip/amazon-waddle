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

type Armazenamento = Pick<Storage, "getItem" | "setItem" | "removeItem">;

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

// A cotação do Frete (5.3), como o Go a devolve em `GET /api/v1/frete`. O
// total vem somado: a tela mostra as três parcelas e não soma nada (NFR-13). A
// Regra de Frete inteira é do Go (AD-17) — aqui não existe região nem valor.
export type Cotacao = {
  regiao: string;
  subtotal_centavos: number;
  frete_centavos: number;
  total_centavos: number;
};

// A rota da cotação para o Endereço escolhido. Perguntada a cada abertura da
// Revisão: é isso que faz a troca de Endereço recalcular o Frete (FR-20).
export function rotaDoFrete(enderecoID: string): string {
  return `/api/v1/frete?endereco_id=${encodeURIComponent(enderecoID)}`;
}

// Frete zero é isenção, e a tela diz "Grátis" em vez de "R$ 0,00". Quem
// decide que é zero é o Go; a tela só escolhe a palavra.
export function freteGratis(cotacao: Pick<Cotacao, "frete_centavos">): boolean {
  return cotacao.frete_centavos === 0;
}

// A chave de idempotência do checkout (5.4, FR-22, NFR-12). É **estado de
// tela**, e não de domínio: o Node continua sem nada de domínio (AD-10). Quem
// a envia no `POST /api/v1/pedidos`, e quem chama `limparCheckout` depois de o
// Pedido nascer, é a 5.6.
//
// Nasce ao **entrar** na Revisão, e não no clique: gerada no clique, o duplo
// clique produziria duas chaves e dois Pedidos, e a idempotência do servidor
// estaria correta e inútil. Fica em `sessionStorage` porque o recarregamento
// da página é justamente o caso que ela existe para proteger.
export const CHAVE_DE_IDEMPOTENCIA = "azamon:checkout:idempotency_key";

// Só o que a geração usa. `randomUUID` é opcional de propósito: em contexto
// não seguro — um IP da LAN na apresentação, que não é `localhost` nem HTTPS —
// o navegador não o expõe, e só `getRandomValues` sobra.
type Cripto = Pick<Crypto, "getRandomValues"> & { randomUUID?: Crypto["randomUUID"] };

const doNavegador = (): Cripto => crypto;

// Um UUIDv4 do próprio navegador. Nada de rede externa (NFR-15) e nada de
// biblioteca: os 16 bytes saem do `crypto`, com a versão no nibble alto do
// sétimo byte e a variante nos dois bits altos do nono. O parâmetro existe
// para o teste montar a queda sem navegador.
export function uuidNovo(cripto: () => Cripto = doNavegador): string {
  const c = cripto();
  if (typeof c.randomUUID === "function") return c.randomUUID();
  const b = c.getRandomValues(new Uint8Array(16));
  b[6] = (b[6] & 0x0f) | 0x40;
  b[8] = (b[8] & 0x3f) | 0x80;
  const h = Array.from(b, (n) => n.toString(16).padStart(2, "0"));
  return [h.slice(0, 4), h.slice(4, 6), h.slice(6, 8), h.slice(8, 10), h.slice(10, 16)]
    .map((parte) => parte.join(""))
    .join("-");
}

// A forma de um UUIDv4: oito-quatro-quatro-quatro-doze em minúscula, com a
// versão `4` e a variante `8`–`b` nos lugares certos.
const FORMA_DO_UUIDV4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

export function pareceUUIDV4(valor: string): boolean {
  return FORMA_DO_UUIDV4.test(valor);
}

// A chave desta tentativa de checkout: a guardada, se já existe — é o que faz
// o recarregamento continuar na mesma —, senão uma nova, guardada. Sem
// armazenamento devolve `null`, e a Revisão avisa em vez de prometer uma
// confirmação que não seria protegida.
//
// O valor guardado é **conferido** antes de ser reaproveitado. O
// `sessionStorage` é do navegador, e qualquer aba, extensão ou dedo no console
// escreve nele: na 5.6 essa string vira o cabeçalho `Idempotency-Key`, onde um
// caractere fora do *token* HTTP, ou um valor quilométrico, quebra a
// requisição em vez de falhar com jeito. Fora da forma, vale como nada
// guardado — gera e sobrescreve.
export function chaveDeIdempotencia(
  armazenamento: () => Armazenamento = daSessao,
  cripto: () => Cripto = doNavegador,
): string | null {
  try {
    const guardada = armazenamento().getItem(CHAVE_DE_IDEMPOTENCIA);
    if (guardada !== null && pareceUUIDV4(guardada)) return guardada;
    const nova = uuidNovo(cripto);
    armazenamento().setItem(CHAVE_DE_IDEMPOTENCIA, nova);
    return nova;
  } catch {
    return null;
  }
}

// O fim da tentativa de checkout: a escolha de Endereço e a chave saem juntas.
// Nasce aqui, mas **quem a chama é a 5.6**, na criação do Pedido — apagar
// antes disso daria ao duplo clique duas chaves, que é o que a chave evita.
// Cada chave no seu `try`: um armazenamento que lança na primeira não pode
// deixar a segunda para trás.
export function limparCheckout(armazenamento: () => Armazenamento = daSessao): void {
  for (const chave of [CHAVE_DO_ENDERECO, CHAVE_DE_IDEMPOTENCIA]) {
    try {
      armazenamento().removeItem(chave);
    } catch {
      // Armazenamento indisponível equivale a nada guardado: não há o que apagar.
    }
  }
}
