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

// O motivo da última entrada em PAGAMENTO_RECUSADO, como a tela o escreve. A
// expiração tem frase própria (EXPERIENCE, "Tentativa expirada"); a recusa do
// Provedor, uma genérica — o motivo que ela grava é da 5.10.
export function motivoDaRecusa(historico: readonly Transicao[]): string {
  for (let i = historico.length - 1; i >= 0; i--) {
    if (historico[i].para === PAGAMENTO_RECUSADO) {
      return historico[i].motivo === TEMPO_ESGOTADO ? "Tempo de pagamento expirado." : "O pagamento foi recusado.";
    }
  }
  return "O pagamento foi recusado.";
}

// As tentativas restantes, com o termo do glossário e a concordância certa.
export function textoDasTentativas(restantes: number): string {
  if (restantes <= 0) return "Não restam Tentativas de Pagamento para este Pedido.";
  if (restantes === 1) return "Resta 1 Tentativa de Pagamento para este Pedido.";
  return `Restam ${restantes} Tentativas de Pagamento para este Pedido.`;
}
