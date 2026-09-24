// A tela do Pedido (5.8). O que ela mostra vem pronto do Go, numa chamada só
// (AD-18): Status, `expira_em`, `tentativas_restantes`, Itens com `disponivel`,
// valores, Endereço e histórico. Aqui mora só a leitura dessa resposta — qual
// superfície abrir, de quanto em quanto consultar, quanto tempo falta e como
// escrever o motivo —, em funções puras para o `node --test` alcançar. Nenhuma
// regra de negócio (AD-10): terminal é o `terminal` do Go, e o prazo é o
// `expira_em` do Go.

import type { Campo } from "./endereco";

export type Transicao = {
  de: string | null;
  para: string;
  ator: string;
  motivo: string | null;
  em: string;
};

export type ItemDoPedido = {
  produto_id: string;
  nome: string;
  quantidade: number;
  preco_praticado_centavos: number;
  disponivel: boolean;
};

export type DetalheDoPedido = {
  id: string;
  numero: string;
  status: string;
  total_centavos: number;
  atualizado_em: string;
  terminal: boolean;
  pode_cancelar: boolean;
  subtotal_centavos: number;
  frete_centavos: number;
  expira_em: string | null;
  tentativas_restantes: number;
  endereco: Record<Campo, string> | null;
  itens: ItemDoPedido[];
  historico: Transicao[];
};

export const AGUARDANDO_PAGAMENTO = "AGUARDANDO_PAGAMENTO";
export const PAGAMENTO_RECUSADO = "PAGAMENTO_RECUSADO";
export const TEMPO_ESGOTADO = "TEMPO_ESGOTADO";

// 3 s enquanto o Pedido aguarda pagamento — a janela em que a confirmação
// chega, e bem menor que os 60 s da demonstração, para a virada parecer
// imediata — e 10 s no resto do caminho, que anda por etapas mais longas
// (EXPERIENCE, Modo demonstração). Sem WebSocket (AD-10).
export const INTERVALO_AGUARDANDO_MS = 3000;
export const INTERVALO_AVANCANDO_MS = 10000;

// As três superfícies da mesma página. `processando` tem o relógio;
// `recusado` troca o relógio pelo motivo, por recusa ou por expiração; e todo
// o resto é o Detalhe, que aparece no lugar, sem navegação nova.
export type Superficie = "processando" | "recusado" | "detalhe";

export function superficieDoPedido(p: Pick<DetalheDoPedido, "status">): Superficie {
  if (p.status === AGUARDANDO_PAGAMENTO) return "processando";
  if (p.status === PAGAMENTO_RECUSADO) return "recusado";
  return "detalhe";
}

// De quanto em quanto consultar de novo, ou null para parar. A condição do
// ritmo rápido é estar em AGUARDANDO_PAGAMENTO, e não "ainda não pago": parar
// ao virar PAGAMENTO_RECUSADO deixaria a tela morta justamente quando a nova
// Tentativa (5.10) pode trazê-la de volta. Quem diz que acabou é o Go.
export function intervaloDaConsulta(p: Pick<DetalheDoPedido, "status" | "terminal">): number | null {
  if (p.terminal) return null;
  return p.status === AGUARDANDO_PAGAMENTO ? INTERVALO_AGUARDANDO_MS : INTERVALO_AVANCANDO_MS;
}

export type TempoRestante = { ms: number; texto: string; esgotado: boolean };

// Quanto falta até `expira_em`, que é instante absoluto do servidor: o
// navegador só subtrai o relógio de agora, então recarregar a página continua
// do mesmo ponto — uma duração recomeçaria (AD-18). Os segundos arredondam
// para cima, para o "0:00" coincidir com o vencimento, e não chegar um
// segundo antes. Instante que não se lê devolve null: melhor não mostrar
// relógio do que mostrar um relógio errado.
export function tempoRestante(expiraEm: string, agoraMs: number): TempoRestante | null {
  const fim = Date.parse(expiraEm);
  if (Number.isNaN(fim)) return null;
  const ms = Math.max(0, fim - agoraMs);
  const segundos = Math.ceil(ms / 1000);
  const texto = `${Math.floor(segundos / 60)}:${String(segundos % 60).padStart(2, "0")}`;
  return { ms, texto, esgotado: ms === 0 };
}

// O motivo que a varredura grava ao aplicar a recusa do Provedor (5.10).
export const RECUSADO_PELO_PROVEDOR = "RECUSADO_PELO_PROVEDOR";

// A frase de cada motivo que o Go grava. Nomear o motivo sempre que o sistema
// souber (EXPERIENCE, Voice and Tone): o Provedor Simulado não tem motivo além
// da própria recusa, e é isso que a frase diz, sem inventar "saldo".
const frasesDoMotivo: Record<string, string> = {
  [TEMPO_ESGOTADO]: "Tempo de pagamento expirado.",
  [RECUSADO_PELO_PROVEDOR]: "O Provedor de Pagamento recusou a Tentativa de Pagamento.",
};

// A frase de um motivo gravado, ou null quando a tela não o conhece. `hasOwn`,
// e não a indexação crua: um motivo como "constructor" acharia o herdado de
// Object.prototype.
export function fraseDoMotivo(motivo: string | null): string | null {
  return motivo !== null && Object.hasOwn(frasesDoMotivo, motivo) ? frasesDoMotivo[motivo] : null;
}

// O motivo da última entrada em PAGAMENTO_RECUSADO, como a tela o escreve. A
// expiração tem frase própria (EXPERIENCE, "Tentativa expirada"), e a recusa
// do Provedor também; motivo que a tela não conhece cai na genérica.
export function motivoDaRecusa(historico: readonly Transicao[]): string {
  for (let i = historico.length - 1; i >= 0; i--) {
    if (historico[i].para === PAGAMENTO_RECUSADO) {
      return fraseDoMotivo(historico[i].motivo) ?? "O pagamento foi recusado.";
    }
  }
  return "O pagamento foi recusado.";
}

// Uma linha da linha do tempo do Detalhe (FR-30, NFR-9): o que aconteceu, na
// ordem em que o Go devolve, e nada do que ainda vai acontecer. O ator não
// sai: é do Administrador e do NFR-9, não do Comprador.
export type LinhaDoTempo = { status: string; antes: string | null; motivo: string | null; em: string };

export function linhaDoTempo(historico: readonly Transicao[]): LinhaDoTempo[] {
  return historico.map((t) => ({
    status: rotuloDoStatus(t.para),
    antes: t.de === null ? null : rotuloDoStatus(t.de),
    motivo: fraseDoMotivo(t.motivo),
    em: t.em,
  }));
}

// Por que não há "Cancelar Pedido" (UX-DR11, ausente com frase): só depois
// que o Pedido saiu para entrega. Nos quatro canceláveis há o botão
// (`podeCancelar`); em CANCELADO não há o que explicar.
const frasesSemCancelamento: Record<string, string> = {
  ENVIADO: "Este Pedido já saiu para entrega e não pode mais ser cancelado.",
  ENTREGUE: "Este Pedido já foi entregue e não pode mais ser cancelado.",
};

export function porQueNaoCancela(status: string): string | null {
  return Object.hasOwn(frasesSemCancelamento, status) ? frasesSemCancelamento[status] : null;
}

// Se o botão "Cancelar Pedido" aparece. A janela (FR-31) é do Go, que manda
// `pode_cancelar` pronto, tirado da tabela do AD-3: uma cópia dos quatro Status
// aqui seria regra de negócio no Node (AD-10), e divergiria em silêncio da
// tabela na primeira mudança. Quem recusa continua sendo a rota.
export function podeCancelar(p: Pick<DetalheDoPedido, "pode_cancelar">): boolean {
  return p.pode_cancelar === true;
}

export function rotaDoCancelamento(pedidoId: string): string {
  return `/api/v1/pedidos/${encodeURIComponent(pedidoId)}/cancelamento`;
}

export const FALHA_NO_CANCELAMENTO = "Não foi possível cancelar o Pedido.";

// O prazo do cancelamento, o mesmo do Confirmar Pedido
// (`PRAZO_DA_CONFIRMACAO_MS`, em lib/checkout.ts). Durante o envio o Dialog não
// fecha, então uma requisição pendurada o prenderia aberto para sempre;
// esgotado o prazo, a tela solta a guarda e mostra a falha no Dialog. Tentar
// de novo é seguro: sobre o Pedido já cancelado, a rota responde 200 sem efeito.
export const PRAZO_DO_CANCELAMENTO_MS = 30_000;

// Os textos do botão e do Dialog de confirmação, que nomeia o Pedido pelo
// número (FR-31). Nenhum nomeia a Reserva de Estoque — o Comprador vê o efeito
// na Vitrine — e nenhum promete reembolso: o MVP não tem.
export function textosDoCancelamento(numero: string) {
  return {
    botao: "Cancelar Pedido",
    titulo: `Cancelar o Pedido ${numero}?`,
    descricao: "Depois de cancelado, o Pedido não volta a andar.",
    manter: "Manter o Pedido",
    confirmar: "Cancelar Pedido",
    enviando: "Cancelando o Pedido…",
  };
}

// A corrida perdida (EXPERIENCE, "cancelamento recusado porque o estado
// mudou"): o Pedido saiu da janela entre a tela e o clique. Ela fica no lugar
// da frase de `porQueNaoCancela`, e não num Toast nem num erro — o Comprador
// não errou nada, e a tela relida já mostra o Status novo.
export const CORRIDA_DO_CANCELAMENTO =
  "Este Pedido foi enviado enquanto você estava nesta tela e não pode mais ser cancelado.";

export type DesfechoDoCancelamento = DesfechoDaNovaTentativa;

// O que fazer com a resposta de `POST .../cancelamento`. O 200 relê: é a
// leitura do Detalhe que traz o Status novo e a linha no histórico — e é 200
// também o segundo clique, sobre o Pedido já cancelado. ESTADO_JA_AVANCADO
// relê calado, como na nova Tentativa: o Pedido já anda, e a tela relida o
// mostra andando. FORA_DA_JANELA_DE_CANCELAMENTO relê e explica.
export function desfechoDoCancelamento(r: { resposta: { ok: boolean; status: number }; json: unknown }): DesfechoDoCancelamento {
  const { ok, status } = r.resposta;
  if (status === 401) return { tipo: "semSessao" };
  if (ok) return { tipo: "releitura", aviso: null };
  const erro = (r.json as { erro?: { codigo?: string; mensagem?: string } } | null)?.erro;
  const codigo = erro?.codigo ?? "";
  if (status === 409 && codigo === "ESTADO_JA_AVANCADO") return { tipo: "releitura", aviso: null };
  if (status === 409 && codigo === "FORA_DA_JANELA_DE_CANCELAMENTO") {
    return { tipo: "releitura", aviso: CORRIDA_DO_CANCELAMENTO };
  }
  return { tipo: "erro", mensagem: erro?.mensagem ?? FALHA_NO_CANCELAMENTO };
}

// A frase de quando não há "Cancelar Pedido": a da corrida perdida, se houve,
// e senão a do Status. Com o botão na tela não há frase nenhuma — a corrida
// só acontece saindo da janela, e de lá o Pedido não volta.
export function fraseSemCancelamento(
  p: Pick<DetalheDoPedido, "status" | "pode_cancelar">,
  avisoDaCorrida: string | null,
): string | null {
  if (podeCancelar(p)) return null;
  return avisoDaCorrida ?? porQueNaoCancela(p.status);
}

// A ação da tripla do AD-18 (EXPERIENCE, "A ação disponível é derivada"):
// "Tentar pagar de novo" existe só com o Pedido recusado, Tentativa restante e
// todos os Itens de Pedido com Estoque disponível. Os três números vêm do Go;
// aqui só se lê. Com o Status sozinho, o botão apareceria para sempre.
export function podeTentarDeNovo(
  p: Pick<DetalheDoPedido, "status" | "tentativas_restantes" | "itens">,
): boolean {
  return p.status === PAGAMENTO_RECUSADO && p.tentativas_restantes > 0 && p.itens.every((item) => item.disponivel);
}

// Por que a ação saiu da tela, quando saiu pelo Estoque: uma frase por Item de
// Pedido sem Estoque disponível, nomeando o Produto (FR-27, "mensagem
// explícita"). Sem Tentativa restante não há frase aqui — `textoDasTentativas`
// já diz que não restam, e repetir seria dizer duas vezes a mesma coisa.
// Produto desativado e esgotado saem iguais, como o Go os devolve (FR-12).
// A chave é o Produto, e não a frase: dois Itens de Pedido podem ter o mesmo
// nome congelado, e a frase repetida seria chave repetida na lista.
export function impedimentosDaNovaTentativa(
  p: Pick<DetalheDoPedido, "status" | "tentativas_restantes" | "itens">,
): { chave: string; texto: string }[] {
  if (p.status !== PAGAMENTO_RECUSADO || p.tentativas_restantes <= 0) return [];
  return p.itens
    .filter((item) => !item.disponivel)
    .map((item) => ({
      chave: item.produto_id,
      texto: `Não há Estoque disponível de ${item.nome} para uma nova Tentativa de Pagamento.`,
    }));
}

export function rotaDaNovaTentativa(pedidoId: string): string {
  return `/api/v1/pedidos/${encodeURIComponent(pedidoId)}/tentativas`;
}

export const FALHA_NA_NOVA_TENTATIVA = "Não foi possível iniciar a nova Tentativa de Pagamento.";

// As recusas da nova Tentativa em que o Pedido é que mudou: sem Estoque, sem
// Tentativa, ou outro caminho chegou antes (inclusive o segundo clique). Nas
// três a resposta certa é reler o Pedido — a tela relida tira o botão e diz
// por quê, pela tripla.
//
// Sem Estoque e sem Tentativa também avisam, com a mensagem do Go: a unidade
// pode ter voltado entre a recusa e a releitura, e aí a tripla relida mostra o
// botão de novo — sem o aviso, o clique pareceria não ter feito nada. A
// corrida não avisa: o Pedido já anda, e a tela relida o mostra andando.
const RECUSAS_QUE_AVISAM = ["ESTOQUE_INSUFICIENTE", "TETO_DE_TENTATIVAS"];
const RECUSAS_CALADAS = ["ESTADO_JA_AVANCADO"];

export type DesfechoDaNovaTentativa =
  | { tipo: "releitura"; aviso: string | null }
  | { tipo: "semSessao" }
  | { tipo: "erro"; mensagem: string };

// O que fazer com a resposta de `POST .../tentativas`. O 201 também relê: é a
// leitura do Detalhe que traz o prazo novo e as Tentativas restantes.
export function desfechoDaNovaTentativa(r: { resposta: { ok: boolean; status: number }; json: unknown }): DesfechoDaNovaTentativa {
  const { ok, status } = r.resposta;
  if (status === 401) return { tipo: "semSessao" };
  if (ok) return { tipo: "releitura", aviso: null };
  const erro = (r.json as { erro?: { codigo?: string; mensagem?: string } } | null)?.erro;
  const codigo = erro?.codigo ?? "";
  if (status === 409 && RECUSAS_CALADAS.includes(codigo)) return { tipo: "releitura", aviso: null };
  if (status === 409 && RECUSAS_QUE_AVISAM.includes(codigo)) {
    return { tipo: "releitura", aviso: erro?.mensagem ?? FALHA_NA_NOVA_TENTATIVA };
  }
  return { tipo: "erro", mensagem: erro?.mensagem ?? FALHA_NA_NOVA_TENTATIVA };
}

// As tentativas restantes, com o termo do glossário e a concordância certa.
export function textoDasTentativas(restantes: number): string {
  if (restantes <= 0) return "Não restam Tentativas de Pagamento para este Pedido.";
  if (restantes === 1) return "Resta 1 Tentativa de Pagamento para este Pedido.";
  return `Restam ${restantes} Tentativas de Pagamento para este Pedido.`;
}

// Os sete Status do CHECK do banco, escritos como se leem em tela. Ponto
// único de rótulo: sem ele, uma renomeação deixaria uma superfície mostrando
// o identificador cru. Aqui mora o texto; a aparência do selo mora logo
// abaixo, em `aparenciaDoSelo` (6.7), e o `SeloDoStatus` só a desenha.
const rotulosDoStatus: Record<string, string> = {
  AGUARDANDO_PAGAMENTO: "Aguardando pagamento",
  PAGAMENTO_RECUSADO: "Pagamento recusado",
  PAGO: "Pago",
  SEPARANDO: "Separando",
  ENVIADO: "Enviado",
  ENTREGUE: "Entregue",
  CANCELADO: "Cancelado",
};

// O rótulo de um Status. Status que a tela não conhece sai cru: um Status novo
// no banco aparece feio, e não invisível. `hasOwn`, e não a indexação crua —
// um "constructor" vindo do servidor acharia o herdado de Object.prototype.
export function rotuloDoStatus(status: string): string {
  return Object.hasOwn(rotulosDoStatus, status) ? rotulosDoStatus[status] : status;
}

// O tom do selo de Status (6.7, UX-DR10). O verde é só de `ENTREGUE`, o
// destrutivo só das duas falhas, e todo o resto — inclusive o Status que a
// tela não conhece — é progresso. O desconhecido nunca sai verde: a cor de fim
// bem-sucedido não se adivinha.
export type TomDoStatus = "progresso" | "falha" | "entregue";

// O mapa é só o das duas falhas. O verde fica fora dele, numa comparação com
// um Status só: uma entrada nova no mapa não tem como ganhar o verde.
const falhasDoStatus: Record<string, true> = {
  PAGAMENTO_RECUSADO: true,
  CANCELADO: true,
};

// `hasOwn` pelo mesmo motivo de `rotuloDoStatus`: um "constructor" vindo do
// servidor não pode achar uma falha herdada.
export function tomDoStatus(status: string): TomDoStatus {
  if (status === "ENTREGUE") return "entregue";
  return Object.hasOwn(falhasDoStatus, status) ? "falha" : "progresso";
}

// A aparência do selo de Status, inteira, como dado: a variante do `Badge` e o
// `className` que o `cn` do `Badge` funde sobre a base. Mora aqui, e não no
// componente, para que o `node --test` prenda classe por classe — e este é o
// único arquivo que pinta com o verde de `ENTREGUE` (a guarda de fonte em
// `scripts/pedido.test.mjs` prende isso).
//
// - Pílula: `rounded-full`, porque o `rounded-4xl` da base está preso em 8px
//   no globals.css.
// - 14px (`text-sm`, altura livre): por decisão humana de 2026-09-24, o
//   rótulo do selo é texto de conteúdo, e o DESIGN.md não o deixa em 12px.
//   O `badge.tsx` do shadcn não é editado; a sobreposição vai no uso.
// - `progresso` é o `secondary` do shadcn, sem cor própria.
// - `falha` é preenchido: o `destructive` do shadcn tinge a 10% e dá 3,99:1,
//   abaixo do AA; o vermelho cheio sob branco dá 4,76:1.
// - `entregue` é o verde do Estoque disponível, o único outro uso dele.
export type AparenciaDoSelo = { variant: "secondary" | "default"; className: string };

const FORMA_DO_SELO = "h-auto rounded-full text-sm";

const classesDoTom: Record<TomDoStatus, string> = {
  progresso: "",
  falha: "bg-destructive text-white",
  entregue: "bg-available text-available-foreground",
};

export function aparenciaDoSelo(status: string): AparenciaDoSelo {
  const tom = tomDoStatus(status);
  const cor = classesDoTom[tom];
  return {
    variant: tom === "progresso" ? "secondary" : "default",
    className: cor ? `${FORMA_DO_SELO} ${cor}` : FORMA_DO_SELO,
  };
}

// Os sete, para o filtro da Tabela do Administrador (6.4). Derivados do mapa
// de rótulos, e não digitados de novo: uma oitava lista divergiria dele.
export const STATUS = Object.keys(rotulosDoStatus);

// ————— O painel de Pedidos do Administrador (6.4, FR-32) —————
//
// Tudo que as duas telas derivam vem pronto do Go, numa chamada só (AD-18):
// `permitidas` — os destinos que o Administrador alcança a partir do Status
// atual — e `terminal`. Nenhuma tabela de transições é redeclarada aqui
// (AD-10); aqui mora só a leitura dessa resposta, em funções puras.

export type PedidoNaTabela = {
  id: string;
  numero: string;
  status: string;
  total_centavos: number;
  criado_em?: string;
  terminal: boolean;
  permitidas: string[];
  // O sinal da FR-26 (6.5): o Provedor aprovou uma Tentativa de Pagamento
  // deste Pedido, que está cancelado — a aprovação pode ter chegado antes do
  // cancelamento (a cobrança duplicada) ou depois, e o texto não diz qual.
  // Derivado no Go (CANCELADO e uma aprovação sinalizada na inbox), sempre
  // presente; a tela só o lê.
  pagamento_aprovado_sobre_cancelado: boolean;
};

// O marcador da linha da Tabela e o Alert persistente do Detalhe, para o
// Pedido com `pagamento_aprovado_sobre_cancelado`. Texto, e não cor: nem o
// verde nem o laranja do DESIGN têm lugar aqui. Nenhuma frase promete estorno
// nem reembolso — a fase 1 não tem nenhum dos dois; o que ela garante é que a
// aprovação não se perde.
export const MARCADOR_APROVADO_SOBRE_CANCELADO = "Pagamento aprovado";
export const TITULO_APROVADO_SOBRE_CANCELADO = "Pagamento aprovado sobre Pedido cancelado";
export const TEXTO_APROVADO_SOBRE_CANCELADO =
  "O Provedor de Pagamento aprovou uma Tentativa de Pagamento deste Pedido cancelado. " +
  "O Status do Pedido não muda, e a aprovação fica registrada na Tentativa de Pagamento.";

export type ItemDoPedidoAdmin = {
  produto_id: string;
  nome: string;
  vendedor_nome: string;
  quantidade: number;
  preco_praticado_centavos: number;
};

export type DetalheAdmin = PedidoNaTabela & {
  atualizado_em: string;
  comprador: { id: string; nome: string; email: string };
  subtotal_centavos: number;
  frete_centavos: number;
  endereco: Record<Campo, string> | null;
  itens: ItemDoPedidoAdmin[];
  historico: Transicao[];
};

// As duas ordenações por data, como a URL e o Go as escrevem.
export const RECENTES = "recentes";
export const ANTIGOS = "antigos";
export const ORDENACOES = [RECENTES, ANTIGOS];

// O rótulo do botão de cada destino. Não é "Marcar como {Status}": o que o
// Administrador faz é a operação, e é ela que o botão nomeia (Voice and Tone).
// Destino que a tela não conhece — uma linha nova na tabela do AD-3 — aparece
// com o rótulo do Status, feio e não invisível, como em `rotuloDoStatus`.
const rotulosDaAcao: Record<string, string> = {
  SEPARANDO: "Iniciar a separação",
  ENVIADO: "Registrar o envio",
  ENTREGUE: "Registrar a entrega",
};

export function rotuloDaAcao(destino: string): string {
  return Object.hasOwn(rotulosDaAcao, destino) ? rotulosDaAcao[destino] : `Mudar para ${rotuloDoStatus(destino)}`;
}

export function rotaDaTransicao(pedidoId: string): string {
  return `/api/v1/admin/pedidos/${encodeURIComponent(pedidoId)}/transicoes`;
}

export const FALHA_NA_TRANSICAO = "Não foi possível mudar o Status do Pedido.";

// O Alert do desfecho, nas duas superfícies. A variante é o que a EXPERIENCE
// separa: a transição fora da tabela é defeito de quem chamou e sai
// `destructive`; o Pedido que outro ator já moveu é informação, e sai neutro —
// dizer "transição inválida" ali reportaria causa falsa.
export type AlertaDaTransicao = { variante: "destrutivo" | "informativo"; texto: string };

export type DesfechoDaTransicao =
  | { tipo: "releitura"; alerta: AlertaDaTransicao | null }
  | { tipo: "semSessao" }
  | { tipo: "erro"; mensagem: string };

// O que o bloco de ação reporta à superfície que o monta. O desfecho NÃO pode
// ficar dentro da linha: com a Tabela filtrada por Status, a releitura tira o
// Pedido da lista e o componente desmonta junto com o Alert — o Administrador
// ficaria sem saber o que aconteceu, que é justamente o Critério de Aceite da
// corrida. Quem renderiza é a superfície, fora da linha.
export type RelatoDaTransicao = { alerta: AlertaDaTransicao | null; erro: string | null };

export const SEM_RELATO: RelatoDaTransicao = { alerta: null, erro: null };

// A frase do Alert destrutivo: nomeia a transição tentada e as permitidas que
// o Go devolveu em `dados.permitidas`. Sem nenhuma permitida — um Pedido
// terminal — a frase diz isso, em vez de terminar numa lista vazia.
function fraseDaTransicaoInvalida(de: string, para: string, permitidas: string[]): string {
  const tentada = `Não é possível ir de ${rotuloDoStatus(de)} para ${rotuloDoStatus(para)}.`;
  if (permitidas.length === 0) return `${tentada} Este Pedido não tem mudanças de Status disponíveis.`;
  return `${tentada} A partir de ${rotuloDoStatus(de)}, só: ${permitidas.map(rotuloDoStatus).join(", ")}.`;
}

// O que fazer com a resposta de `POST .../transicoes`. O 200 relê: é a leitura
// que traz o Status novo, o `permitidas` novo e a linha do histórico. As duas
// recusas da máquina também releem — nas duas o Pedido não é mais o que a tela
// mostrava —, cada uma com o seu Alert. 404 é o do prefixo administrativo: a
// Sessão acabou, e a tela vai ao login (o mesmo desvio de `/admin/produtos`).
export function desfechoDaTransicao(
  r: { resposta: { ok: boolean; status: number }; json: unknown },
  tentada: { de: string; para: string },
): DesfechoDaTransicao {
  const { ok, status } = r.resposta;
  if (status === 404) return { tipo: "semSessao" };
  if (ok) return { tipo: "releitura", alerta: null };
  const erro = (r.json as { erro?: { codigo?: string; mensagem?: string; dados?: Record<string, unknown> } } | null)?.erro;
  const codigo = erro?.codigo ?? "";
  const dados = erro?.dados ?? {};
  if (status === 409 && codigo === "TRANSICAO_INVALIDA") {
    const permitidas = Array.isArray(dados.permitidas) ? (dados.permitidas as string[]) : [];
    return {
      tipo: "releitura",
      alerta: { variante: "destrutivo", texto: fraseDaTransicaoInvalida(tentada.de, tentada.para, permitidas) },
    };
  }
  if (status === 409 && codigo === "ESTADO_JA_AVANCADO") {
    const atual = typeof dados.status === "string" ? dados.status : "";
    return {
      tipo: "releitura",
      alerta: {
        variante: "informativo",
        texto: `Este Pedido já está em ${rotuloDoStatus(atual)}. A linha foi atualizada.`,
      },
    };
  }
  return { tipo: "erro", mensagem: erro?.mensagem ?? FALHA_NA_TRANSICAO };
}

// O estado inteiro da Tabela mora na URL: recarregar, voltar pelo histórico do
// navegador ou compartilhar o endereço reproduzem a mesma lista.
export type FiltroDaTabela = { status: string; ordenacao: string; pagina: number };

// O que a URL diz, com o que ela não diz valendo o padrão. Valor fora das
// listas fechadas cai no padrão em vez de viajar ao Go para levar 400: o que
// chega aqui é o que o usuário digitou na barra de endereços.
export function filtroDaURL(params: { get(chave: string): string | null }): FiltroDaTabela {
  const status = params.get("status") ?? "";
  const ordenacao = params.get("ordenacao") ?? "";
  const pagina = Number.parseInt(params.get("pagina") ?? "1", 10);
  return {
    status: STATUS.includes(status) ? status : "",
    ordenacao: ORDENACOES.includes(ordenacao) ? ordenacao : RECENTES,
    pagina: Number.isSafeInteger(pagina) ? Math.max(1, pagina) : 1,
  };
}

// A query da URL e a da API saem da mesma função: uma segunda montagem
// divergiria, e a lista exibida deixaria de ser a que o endereço nomeia. O
// padrão fica de fora da URL — `?ordenacao=recentes&pagina=1` é ruído.
export function consultaDaTabela(f: FiltroDaTabela): string {
  const q = new URLSearchParams();
  if (f.status) q.set("status", f.status);
  if (f.ordenacao !== RECENTES) q.set("ordenacao", f.ordenacao);
  if (f.pagina > 1) q.set("pagina", String(f.pagina));
  return q.toString();
}

export function enderecoDaTabela(f: FiltroDaTabela): string {
  const q = consultaDaTabela(f);
  return q ? `/admin/pedidos?${q}` : "/admin/pedidos";
}

export function rotaDaTabela(f: FiltroDaTabela): string {
  const q = consultaDaTabela(f);
  return q ? `/api/v1/admin/pedidos?${q}` : "/api/v1/admin/pedidos";
}

// De quanto em quanto reler a Tabela, ou null para parar: 10 s enquanto a
// página listar Pedido não terminal. Quem diz que acabou é o `terminal` do Go
// — uma cópia de "ENTREGUE ou CANCELADO" aqui divergiria da máquina.
export function intervaloDaTabela(itens: readonly Pick<PedidoNaTabela, "terminal">[]): number | null {
  return itens.some((p) => !p.terminal) ? INTERVALO_AVANCANDO_MS : null;
}

// O anúncio da consulta em intervalo: só os Pedidos cujo Status mudou desde a
// leitura anterior, cada um nomeado pelo número. Uma região viva para a Tabela
// inteira, e não uma por linha — uma por linha anunciaria "Separando" sem
// dizer de qual Pedido, e criaria tantas regiões quantas forem as linhas.
// Vazio quando nada mudou, para o leitor de tela não reler o que já leu; e
// Pedido que ainda não estava na leitura anterior não é mudança, é chegada.
export function anuncioDaTabela(
  anteriores: Record<string, string>,
  itens: readonly Pick<PedidoNaTabela, "id" | "numero" | "status">[],
): string {
  return itens
    .filter((p) => Object.hasOwn(anteriores, p.id) && anteriores[p.id] !== p.status)
    .map((p) => `Pedido ${p.numero}: ${rotuloDoStatus(p.status)}.`)
    .join(" ");
}

// Os Status da leitura atual, para a próxima comparar com eles.
export function statusPorPedido(
  itens: readonly Pick<PedidoNaTabela, "id" | "status">[],
): Record<string, string> {
  return Object.fromEntries(itens.map((p) => [p.id, p.status]));
}
