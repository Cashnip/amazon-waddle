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

// O motivo da última entrada em PAGAMENTO_RECUSADO, como a tela o escreve. A
// expiração tem frase própria (EXPERIENCE, "Tentativa expirada"), e a recusa
// do Provedor também; motivo que a tela não conhece cai na genérica.
export function motivoDaRecusa(historico: readonly Transicao[]): string {
  for (let i = historico.length - 1; i >= 0; i--) {
    if (historico[i].para === PAGAMENTO_RECUSADO) {
      // `hasOwn`, e não a indexação crua: um motivo como "constructor"
      // acharia o herdado de Object.prototype em vez de cair na genérica.
      const motivo = historico[i].motivo ?? "";
      return Object.hasOwn(frasesDoMotivo, motivo) ? frasesDoMotivo[motivo] : "O pagamento foi recusado.";
    }
  }
  return "O pagamento foi recusado.";
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
// o identificador cru. Aqui mora só o texto — o selo (cor e forma) é da 6.7,
// que o fecha nas três superfícies de uma vez.
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
