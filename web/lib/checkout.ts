// A escolha de Endereço do checkout (5.2). O navegador guarda só o `id` do
// Endereço — nunca Frete, CEP ou total. É a pré-condição para trocar o
// Endereço recalcular o Frete: sem cópia no navegador, a Revisão terá de
// perguntar ao Go. O Frete em si é da 5.3 e da 5.4.
//
// Fica em `sessionStorage`, e não na URL, porque tem de atravessar a ida ao
// Carrinho e a volta (FR-22), e o botão do Carrinho não sabe o `id`. Toda
// leitura e escrita está em `try/catch`: janela privada, armazenamento
// bloqueado ou cota cheia equivalem a nada guardado.

// Só o tipo: a forma da resposta do Go, e nenhuma regra (AD-10). A decisão de
// bloquear ou avisar continua sendo do Go, em `carrinho`.
import type { Carrinho } from "./carrinho";

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

// Para onde a Revisão manda o Comprador, e `null` quando ela pode abrir (5.5).
// Lê o Carrinho que a própria Revisão já pediu — `GET /api/v1/carrinho` traz
// `bloqueio` e `preco_mudou` — e **não escreve nada**: quem reporta e confirma
// é a entrada no checkout, `POST /api/v1/checkout/entrada`, que o passo
// Endereço chama (AD-17). Duas escritas seriam dois lugares gravando o mesmo
// campo.
//
// Bloqueio devolve ao Carrinho, que é onde estão Ajustar e Remover: não há
// caminho até o Confirmar Pedido com uma linha bloqueada (FR-19). Um preço
// que mudou **depois** da entrada devolve ao passo Endereço, que é a rota que
// reporta o "de X para Y" e grava a ciência — na Revisão o aviso não teria
// como ser confirmado. O bloqueio vem primeiro porque confirmar preço não
// desbloqueia: as duas listas são independentes.
//
// Carrinho vazio não é caso desta função: quem junta as duas guardas, e fixa a
// ordem entre elas, é `desvioDaAberturaDaRevisao`.
export function desvioDaRevisao(carrinho: Carrinho): string | null {
  if (carrinho.itens.some((i) => i.bloqueio !== "")) return "/carrinho";
  if (carrinho.itens.some((i) => i.visivel && i.preco_mudou)) return "/checkout/endereco";
  return null;
}

// A decisão inteira da abertura da Revisão sobre o Carrinho, numa função só, e
// **nesta ordem**: Carrinho vazio primeiro, revalidação depois. A ordem mora
// aqui, e não dentro de um `useEffect`, porque é ela que um teste consegue
// prender — no efeito, inverter as duas linhas passaria por toda a suíte.
//
// Vazio vem antes porque um Carrinho sem Item nenhum não tem bloqueio nem
// preço a discutir: mandá-lo ao passo Endereço seria um beco. Depois disso,
// quem decide é `desvioDaRevisao`.
export function desvioDaAberturaDaRevisao(carrinho: Carrinho): string | null {
  if (carrinho.itens.length === 0) return "/carrinho";
  return desvioDaRevisao(carrinho);
}

// O que a entrada no checkout é, escrito num lugar só: **`POST`**, e numa rota
// que **escreve**. O passo Endereço abre por aqui, e não por `GET
// /api/v1/carrinho` — a diferença entre as duas é a estória inteira (AD-17), e
// trocar uma pela outra por engano é justamente o que um teste sobre esta
// constante impede de passar batido.
export const ENTRADA_NO_CHECKOUT = { rota: "/api/v1/checkout/entrada", metodo: "POST" } as const;

// Só o que estas funções leem de uma resposta do Go. O teste monta a sua sem
// navegador, e nenhuma delas toca `fetch`, `window` ou `router`.
export type RespostaDoGo = {
  resposta: { ok: boolean; status: number };
  json: unknown;
};

// A mensagem do envelope de erro (AD-14), com a queda de quem chama quando a
// resposta não trouxe envelope nenhum.
function mensagemDoEnvelope(json: unknown, padrao: string): string {
  const envelope = json as { erro?: { mensagem?: string } } | null;
  return envelope?.erro?.mensagem ?? padrao;
}

// O que a abertura de um passo do checkout decide. `carrinho` viaja em **toda**
// saída a partir do momento em que a entrada respondeu: o relatório já foi
// gravado no banco, e perdê-lo por causa de uma falha posterior seria a mesma
// mudança de preço sumindo sem ninguém ver que o AD-17 proíbe — só que do lado
// do cliente. Por isso `erro` e `carrinho` convivem: o aviso da lista de
// Endereços não apaga os dois `Alert` da revalidação.
export type Abertura<E> = {
  carrinho: Carrinho | null;
  // A Sessão caiu: quem chama manda ao Login com o `destino`, que depende da
  // janela e por isso não é decidido aqui.
  semSessao: boolean;
  destino: string | null;
  erro: string | null;
  enderecos: E[] | null;
};

const ABERTURA_VAZIA = { carrinho: null, semSessao: false, destino: null, erro: null, enderecos: null };

// A abertura do passo Endereço (5.5): a resposta da entrada no checkout e a da
// lista de Endereços, e o que fazer com as duas.
//
// A ordem das guardas é a regra que esta função existe para prender: 401 →
// entrada recusada → **aplicar o relatório** → Carrinho vazio → lista de
// Endereços. Aplicar o relatório antes das duas últimas é o que garante que
// uma ciência já comitada seja mostrada, mesmo quando a lista falha logo
// depois.
export function aberturaDoPassoEndereco<E>(entrada: RespostaDoGo, lista: RespostaDoGo): Abertura<E> {
  if (entrada.resposta.status === 401 || lista.resposta.status === 401) {
    return { ...ABERTURA_VAZIA, semSessao: true };
  }
  if (!entrada.resposta.ok || entrada.json === null) {
    return { ...ABERTURA_VAZIA, erro: mensagemDoEnvelope(entrada.json, "Não foi possível abrir o Carrinho.") };
  }
  // A partir daqui a entrada comitou: o relatório sai em toda saída.
  const carrinho = entrada.json as Carrinho;
  if (carrinho.itens.length === 0) {
    return { ...ABERTURA_VAZIA, carrinho, destino: "/carrinho" };
  }
  if (!lista.resposta.ok) {
    return { ...ABERTURA_VAZIA, carrinho, erro: mensagemDoEnvelope(lista.json, "Não foi possível ler os Endereços.") };
  }
  return { ...ABERTURA_VAZIA, carrinho, enderecos: (lista.json as E[] | null) ?? [] };
}

// A abertura da Revisão, até onde o Carrinho decide. A Revisão **não escreve**
// (AD-17): ela lê o `GET /api/v1/carrinho` e desvia. O desvio vem **antes** da
// guarda da lista de Endereços, e antes de qualquer pintura — quem digitou
// esta URL por cima de um avanço bloqueado não vê a Revisão piscar.
//
// O que sobra depois disto — o Endereço escolhido e a cotação do Frete — segue
// em `enderecoEscolhido` e `rotaDoFrete`, que já eram puros e já têm teste.
export function aberturaDaRevisao<E>(oCarrinho: RespostaDoGo, aLista: RespostaDoGo): Abertura<E> {
  if (oCarrinho.resposta.status === 401 || aLista.resposta.status === 401) {
    return { ...ABERTURA_VAZIA, semSessao: true };
  }
  if (!oCarrinho.resposta.ok || oCarrinho.json === null) {
    return { ...ABERTURA_VAZIA, erro: mensagemDoEnvelope(oCarrinho.json, "Não foi possível abrir o Carrinho.") };
  }
  const carrinho = oCarrinho.json as Carrinho;
  const destino = desvioDaAberturaDaRevisao(carrinho);
  if (destino !== null) {
    return { ...ABERTURA_VAZIA, destino };
  }
  if (!aLista.resposta.ok) {
    return { ...ABERTURA_VAZIA, erro: mensagemDoEnvelope(aLista.json, "Não foi possível ler os Endereços.") };
  }
  return { ...ABERTURA_VAZIA, carrinho, enderecos: (aLista.json as E[] | null) ?? [] };
}

// Se o Continuar do passo Endereço pode disparar, e por que não. As razões são
// texto de tela, e a decisão é do Go — `podeAvancar` vem de `revalidacaoDe`,
// que lê `bloqueio` e `preco_mudou` da resposta.
//
// Fica aqui, e não dentro do componente, pelo mesmo motivo da ordem acima:
// trocar `!marcado && !podeAvancar` por `!marcado` sozinho é uma linha, e sem
// teste ninguém a pega. Confirmar preço **não** desbloqueia: as duas listas
// entram separadas, e as duas razões aparecem quando as duas valem.
export function razoesDoContinuar(bloqueados: number, aConfirmar: number): string[] {
  return [
    ...(bloqueados > 0 ? ["Um Produto do Carrinho precisa de ajuste antes do Pedido."] : []),
    ...(aConfirmar > 0
      ? [aConfirmar === 1 ? "Confirme o novo preço para continuar." : "Confirme os novos preços para continuar."]
      : []),
  ];
}

// O Continuar só dispara com um Endereço marcado E com o Carrinho liberado.
export function podeContinuar(marcado: string | null, podeAvancar: boolean): boolean {
  return marcado !== null && podeAvancar;
}

// A chave de idempotência do checkout (5.4, FR-22, NFR-12). É **estado de
// tela**, e não de domínio: o Node continua sem nada de domínio (AD-10). A
// Revisão a envia no `POST /api/v1/pedidos` (`requisicaoDeConfirmar`) e chama
// `limparCheckout` quando o Pedido nasce (`executarDesfecho`).
//
// Nasce ao **entrar** na Revisão, e não no clique: gerada no clique, o duplo
// clique produziria duas chaves e dois Pedidos, e a idempotência do servidor
// estaria correta e inútil. Fica em `sessionStorage` porque o recarregamento
// da página é justamente o caso que ela existe para proteger. Vale **uma
// tentativa de checkout**: o "Fechar o Pedido" do Carrinho começa outra e a
// descarta (`iniciarTentativaDeCheckout`, 5.6).
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
// Quem a chama é a Revisão, **só quando o Pedido nasceu** — apagar antes disso
// daria ao duplo clique duas chaves, que é o que a chave evita; e recusa por
// Estoque, por total ou por Carrinho não apaga, porque não consome a chave.
// Cada chave no seu `try`: um armazenamento que lança na primeira não pode
// deixar a segunda para trás.
export function limparCheckout(armazenamento: () => Armazenamento = daSessao): void {
  apagar([CHAVE_DO_ENDERECO, CHAVE_DE_IDEMPOTENCIA], armazenamento);
}

// Só a chave: a escolha de Endereço continua. É o que a chave reaproveitada
// pede — ela já criou um Pedido e não serve para outro — e o que começa uma
// tentativa nova de checkout.
export function descartarChave(armazenamento: () => Armazenamento = daSessao): void {
  apagar([CHAVE_DE_IDEMPOTENCIA], armazenamento);
}

// Só a escolha de Endereço: a chave continua. É o 404 da confirmação — o
// Endereço escolhido não é mais do Comprador, e o passo Endereço escolhe outro.
export function descartarEscolha(armazenamento: () => Armazenamento = daSessao): void {
  apagar([CHAVE_DO_ENDERECO], armazenamento);
}

function apagar(chaves: string[], armazenamento: () => Armazenamento): void {
  for (const chave of chaves) {
    try {
      armazenamento().removeItem(chave);
    } catch {
      // Armazenamento indisponível equivale a nada guardado: não há o que apagar.
    }
  }
}

// O começo de uma tentativa de checkout: o "Fechar o Pedido" do Carrinho. A
// chave antiga sai, e a Revisão gera outra ao abrir; a escolha de Endereço
// fica, porque voltar ao Carrinho não pode perdê-la (FR-22).
//
// Rodar a chave aqui é inofensivo porque ela só se prende quando um Pedido
// **comita** (5.6): enquanto nenhum Pedido nasceu, chave nova e chave velha
// valem o mesmo. E é necessário porque o 201 pode se perder no caminho — o
// Pedido nasceu, a tela nunca soube, a chave ficou guardada —, e um checkout
// seguinte sob a mesma chave devolveria em silêncio o Pedido **antigo** (200,
// mesmo total) ou abriria o antigo pelo 409. Dentro da mesma tentativa — o
// recarregamento da Revisão, a ida ao passo Endereço e a volta — a chave
// continua a mesma, que é o caso que ela existe para proteger.
export function iniciarTentativaDeCheckout(armazenamento: () => Armazenamento = daSessao): void {
  descartarChave(armazenamento);
}

// O prazo do Confirmar Pedido. Uma requisição pendurada não pode travar o
// botão para sempre; esgotado o prazo, a tela solta a guarda e a mesma chave
// torna o novo clique seguro — se o Pedido nasceu, o reenvio devolve o mesmo.
export const PRAZO_DA_CONFIRMACAO_MS = 30_000;

// O Confirmar Pedido (5.6), escrito num lugar só: a rota, o método, o
// cabeçalho `Idempotency-Key` e o corpo. O corpo é o Endereço escolhido e o
// total que a Revisão **exibiu** — o da cotação do Go, nunca uma soma feita
// aqui (NFR-13). Quem compara o total e recusa com `TOTAL_DIVERGENTE` é o Go;
// o navegador só o repete.
export function requisicaoDeConfirmar(
  chave: string,
  enderecoID: string,
  totalCentavos: number,
  sinal?: AbortSignal,
): { rota: string; init: RequestInit } {
  return {
    rota: "/api/v1/pedidos",
    init: {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json", "Idempotency-Key": chave },
      body: JSON.stringify({ endereco_id: enderecoID, total_centavos: totalCentavos }),
      signal: sinal,
    },
  };
}

// O que a Revisão faz com a resposta do Confirmar Pedido. Cada recusa tem o
// seu tratamento, e nenhuma cai no aviso genérico quando tem um nome:
//
// - `pedido`: o Pedido nasceu agora (201) ou é o reenvio desta tentativa
//   (200). A tentativa de checkout termina, e a tela vai ao Pedido.
// - `chaveReutilizada`: a chave já tinha criado **outro** Pedido (409
//   `CHAVE_REUTILIZADA`). A chave sai sempre — não serve mais para nada —, e a
//   tela diz que esta confirmação já criou um Pedido, com o link quando o Go
//   mandou o `id`. Não navega em silêncio: o Comprador precisa saber que o
//   Carrinho de agora não virou Pedido.
// - `releitura`: o total mudou desde a Revisão (`TOTAL_DIVERGENTE`). A tela
//   avisa e relê a Revisão, com a **mesma** chave — a recusa não a consumiu.
// - `desvio`: o Endereço escolhido não é mais do Comprador (404
//   `NAO_ENCONTRADO`). A escolha sai, e o passo Endereço escolhe outro.
// - `semSessao`: ao Login, com o `destino` que quem chama monta.
// - `erro`: o resto, com a mensagem do envelope; `comCarrinho` quando a saída
//   é o Carrinho (Estoque, Carrinho vazio ou mudado).
export type Desfecho =
  | { tipo: "pedido"; destino: string }
  | { tipo: "chaveReutilizada"; mensagem: string; pedido: string | null }
  | { tipo: "releitura"; aviso: string }
  | { tipo: "desvio"; destino: string }
  | { tipo: "semSessao" }
  | { tipo: "erro"; mensagem: string; comCarrinho: boolean };

export const FALHA_NA_CONFIRMACAO = "Não foi possível confirmar o Pedido.";
const CHAVE_JA_USADA = "Esta confirmação já criou um Pedido.";
const RECUSAS_DO_CARRINHO = ["ESTOQUE_INSUFICIENTE", "CARRINHO_VAZIO", "CARRINHO_MUDOU"];

const rotaDoPedido = (id: string) => `/pedidos/${encodeURIComponent(id)}`;

export function desfechoDaConfirmacao(r: RespostaDoGo): Desfecho {
  const { status } = r.resposta;
  if (status === 401) return { tipo: "semSessao" };
  if (r.resposta.ok) {
    const id = (r.json as { id?: unknown } | null)?.id;
    // Sem `id` não há para onde levar, mesmo com 2xx: cair calado deixaria a
    // tela sem confirmação e sem erro, e o Comprador confirmaria de novo.
    if (typeof id === "string" && id !== "") return { tipo: "pedido", destino: rotaDoPedido(id) };
    return { tipo: "erro", mensagem: FALHA_NA_CONFIRMACAO, comCarrinho: false };
  }
  const erro = (r.json as { erro?: { codigo?: string; mensagem?: string; dados?: unknown } } | null)?.erro;
  const mensagem = erro?.mensagem ?? FALHA_NA_CONFIRMACAO;
  if (status === 409 && erro?.codigo === "CHAVE_REUTILIZADA") {
    const id = (erro.dados as { id?: unknown } | null)?.id;
    return {
      tipo: "chaveReutilizada",
      mensagem: erro.mensagem ?? CHAVE_JA_USADA,
      pedido: typeof id === "string" && id !== "" ? rotaDoPedido(id) : null,
    };
  }
  if (status === 409 && erro?.codigo === "TOTAL_DIVERGENTE") return { tipo: "releitura", aviso: mensagem };
  // Só o 404 com o código do Go é o Endereço que deixou de ser do Comprador.
  // Outro 404 — um proxy, uma rota que sumiu — é falha, e não motivo para
  // apagar a escolha de ninguém.
  if (status === 404 && erro?.codigo === "NAO_ENCONTRADO") return { tipo: "desvio", destino: "/checkout/endereco" };
  return {
    tipo: "erro",
    mensagem,
    comCarrinho: status === 409 && RECUSAS_DO_CARRINHO.includes(erro?.codigo ?? ""),
  };
}

// O aviso que a Revisão mostra ao lado do botão. `pedido` é o link "Ver o
// Pedido", quando há um Pedido a mostrar.
export type AvisoDaConfirmacao = { mensagem: string; comCarrinho: boolean; pedido: string | null };

// O que a tela sabe fazer, injetado: o teste troca cada um por um registro.
export type EfeitosDaConfirmacao = {
  limpar: () => void;
  descartarChave: () => void;
  descartarEscolha: () => void;
  avisarCarrinho: () => void;
  navegar: (destino: string) => void;
  irAoLogin: () => void;
  reler: () => void;
  mostrarAviso: (aviso: AvisoDaConfirmacao) => void;
};

// Executa o desfecho. Devolve se a guarda de envio pode ser solta: quem
// navega mantém o botão travado até a tela trocar — reabri-lo daria a janela
// de um segundo envio enquanto a tela ainda é esta.
//
// A decisão de **o que** apagar mora aqui, e não no componente, porque é ela
// que um teste consegue prender: a chave só sai quando um Pedido existe ou
// quando ela já não serve (reaproveitada), e nunca numa recusa que não a
// consumiu.
export function executarDesfecho(d: Desfecho, e: EfeitosDaConfirmacao): boolean {
  switch (d.tipo) {
    case "pedido":
      e.limpar();
      e.avisarCarrinho();
      e.navegar(d.destino);
      return false;
    case "chaveReutilizada":
      e.descartarChave();
      e.mostrarAviso({ mensagem: d.mensagem, comCarrinho: false, pedido: d.pedido });
      e.reler();
      return true;
    case "releitura":
      e.mostrarAviso({ mensagem: d.aviso, comCarrinho: false, pedido: null });
      e.reler();
      return true;
    case "desvio":
      e.descartarEscolha();
      e.navegar(d.destino);
      return false;
    case "semSessao":
      e.irAoLogin();
      return false;
    case "erro":
      e.mostrarAviso({ mensagem: d.mensagem, comCarrinho: d.comCarrinho, pedido: null });
      return true;
  }
}
