"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Separator } from "@/components/ui/separator";
import { EnderecoPorExtenso } from "@/components/formulario-de-endereco";
import { Preco } from "@/components/preco";
import { SeloDoStatus } from "@/components/selo-do-status";
import { freteGratis } from "@/lib/checkout";
import { paraLogin } from "@/lib/destino";
import {
  FALHA_NA_NOVA_TENTATIVA,
  FALHA_NO_CANCELAMENTO,
  INTERVALO_AGUARDANDO_MS,
  PAGAMENTO_RECUSADO,
  PRAZO_DO_CANCELAMENTO_MS,
  desfechoDaNovaTentativa,
  desfechoDoCancelamento,
  fraseSemCancelamento,
  impedimentosDaNovaTentativa,
  intervaloDaConsulta,
  linhaDoTempo,
  motivoDaRecusa,
  podeCancelar,
  podeTentarDeNovo,
  rotaDaNovaTentativa,
  rotaDoCancelamento,
  superficieDoPedido,
  tempoRestante,
  textoDasTentativas,
  textosDoCancelamento,
  type DetalheDoPedido,
} from "@/lib/pedido";
import { formatarPreco } from "@/lib/preco";

// A tela do Pedido (5.8) — "Pedido em processamento" enquanto aguarda
// pagamento, apontada pelo §14 do PRD como o elemento de interface mais
// arriscado do documento. É estado próprio, não modal e não spinner, e tem
// três superfícies na mesma página: o relógio enquanto aguarda; o motivo no
// lugar do relógio quando recusa ou expira; e o Detalhe quando o pagamento
// passa, sem navegação nova. "Ver o Pedido em Meus pedidos" existe em todas,
// inclusive carregando e com erro: o Comprador nunca fica preso aqui.
//
// Tudo sai de uma consulta só (AD-18). O que ler dela — superfície, ritmo,
// tempo restante, motivo — mora em `lib/pedido.ts`, testado; aqui só se
// executa.

// O instante vem absoluto e em RFC 3339 do servidor; o `dateTime` guarda essa
// forma e o que o Comprador lê é a mesma marca em pt-BR. É o sinal, durante a
// consulta em intervalo, de que o que está na tela é recente.
function Instante({ valor }: { valor: string }) {
  const data = new Date(valor);
  const texto = Number.isNaN(data.getTime())
    ? valor
    : data.toLocaleString("pt-BR", { dateStyle: "short", timeStyle: "medium" });
  return <time dateTime={valor}>{texto}</time>;
}

// O relógio re-renderiza a cada segundo, mas não conta nada: cada tique só
// subtrai o agora do `expira_em` do servidor, então recarregar a página
// continua do mesmo ponto. O número fica fora de região viva — anunciar um
// valor a cada segundo afogaria o leitor de tela —, mas a frase do prazo
// encerrado entra numa região que existe desde a montagem, para ser anunciada
// uma vez quando aparece.
function Relogio({ expiraEm }: { expiraEm: string }) {
  const [agora, setAgora] = useState(() => Date.now());
  useEffect(() => {
    const tique = setInterval(() => setAgora(Date.now()), 1000);
    return () => clearInterval(tique);
  }, []);

  const restante = tempoRestante(expiraEm, agora);
  if (restante === null) return null;
  return (
    <>
      {!restante.esgotado && (
        <p className="text-sm">
          Tempo restante para o pagamento:{" "}
          {/* `dateTime` de duração (ISO 8601), porque o texto é o que falta,
              e não o instante do vencimento. */}
          <time dateTime={`PT${Math.ceil(restante.ms / 1000)}S`} className="font-medium tabular-nums">
            {restante.texto}
          </time>
        </p>
      )}
      {/* Zerado e ainda aguardando: quem decide o desfecho é o servidor, e a
          consulta segue. A frase diz o que aconteceu e não promete quando o
          Pedido muda — um relógio que para sem dizer por quê é exatamente o
          risco do §14, e uma promessa que não se cumpre é o outro. */}
      <p className="text-sm" role="status">
        {restante.esgotado && "O prazo para o pagamento terminou sem a confirmação chegar."}
      </p>
    </>
  );
}

// A linha do tempo (6.2, FR-30): uma linha por transição registrada, na ordem
// do Go, e nenhuma etapa futura. Marcador de 8 px em `border` ligado por
// filete de 1 px (DESIGN.md); a última linha não tem filete adiante, e é isso
// que fecha ENTREGUE e CANCELADO sem caso especial. Sem cor e sem ícone. A
// mudança de Status já é anunciada pela região viva do topo: aqui não há outra.
function LinhaDoTempo({ pedido }: { pedido: DetalheDoPedido }) {
  const linhas = linhaDoTempo(pedido.historico);
  return (
    <ol className="rounded-md">
      {linhas.map((linha, i) => (
        // O histórico só cresce no fim, então a posição é chave estável.
        <li key={i} className="relative pb-4 pl-5 last:pb-0">
          <span aria-hidden="true" className="bg-background absolute top-1.5 left-0 size-2 rounded-full border" />
          {i < linhas.length - 1 && (
            <span aria-hidden="true" className="bg-border absolute top-3.5 -bottom-1.5 left-[3.5px] w-px" />
          )}
          <p className="font-medium">{linha.status}</p>
          {linha.antes && <p className="text-muted-foreground text-sm">antes: {linha.antes}</p>}
          {linha.motivo && <p className="text-sm">{linha.motivo}</p>}
          <p className="text-muted-foreground text-sm">
            <Instante valor={linha.em} />
          </p>
        </li>
      ))}
    </ol>
  );
}

// O Detalhe do Pedido: Itens com o preço praticado, a linha do tempo, o
// Endereço e os valores congelados. O cancelamento não mora aqui: em
// AGUARDANDO_PAGAMENTO o Detalhe não aparece, e o botão faltaria num dos quatro
// Status da janela — ele fica no Card do topo, junto do Status.
function Detalhe({ pedido }: { pedido: DetalheDoPedido }) {
  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>
            <h2>Itens do Pedido</h2>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ul>
            {pedido.itens.map((item, i) => (
              <li key={item.produto_id}>
                {i > 0 && <Separator className="my-4" />}
                <p className="font-medium">{item.nome}</p>
                <p className="text-muted-foreground text-sm tabular-nums">
                  {item.quantidade} × {formatarPreco(item.preco_praticado_centavos)}
                </p>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>
            <h2>Histórico</h2>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <LinhaDoTempo pedido={pedido} />
        </CardContent>
      </Card>

      {pedido.endereco && (
        <Card>
          <CardHeader>
            <CardTitle>
              <h2>Endereço de entrega</h2>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <EnderecoPorExtenso endereco={pedido.endereco} />
          </CardContent>
        </Card>
      )}

      {/* Os três valores são os congelados no Pedido, como o Go os devolve:
          a tela não soma nada (NFR-13), e o total fecha com as parcelas
          porque o banco o exige (AD-9). */}
      <Card>
        <CardHeader>
          <CardTitle>
            <h2>Valores</h2>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="space-y-2 text-sm">
            <div className="flex justify-between gap-4">
              <dt>Subtotal</dt>
              <dd className="tabular-nums">{formatarPreco(pedido.subtotal_centavos)}</dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt>Frete</dt>
              <dd className="tabular-nums">
                {freteGratis(pedido) ? "Grátis" : formatarPreco(pedido.frete_centavos)}
              </dd>
            </div>
            <div className="flex justify-between gap-4 border-t pt-2 font-medium">
              <dt>Total</dt>
              <dd className="tabular-nums">{formatarPreco(pedido.total_centavos)}</dd>
            </div>
          </dl>
        </CardContent>
      </Card>
    </>
  );
}

export function Acompanhamento({ pedidoId }: { pedidoId: string }) {
  const router = useRouter();
  const [pedido, setPedido] = useState<DetalheDoPedido | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  // A consulta mora dentro do efeito, com o intervalo dela; a nova Tentativa
  // precisa pedir uma leitura agora, e é por aqui que a alcança.
  const consultarAgora = useRef<() => Promise<void>>(async () => {});
  // Um envio por clique (FR-27): o ref barra o segundo antes de o estado
  // chegar à tela. O servidor também barra — o segundo sai ESTADO_JA_AVANCADO —,
  // mas a tela não precisa gastar a ida.
  const emVoo = useRef(false);
  const [enviando, setEnviando] = useState(false);
  const [erroDaTentativa, setErroDaTentativa] = useState<string | null>(null);
  // Depois da nova Tentativa o botão some com a superfície recusada, e o foco
  // cairia no `body`; ele vai para o título do Pedido, cujo Status a região
  // viva logo abaixo anuncia.
  const titulo = useRef<HTMLHeadingElement>(null);

  // O cancelamento (6.3, FR-31): o Dialog é controlado e vive fora do botão,
  // porque o botão some assim que a releitura traz CANCELADO — e o Dialog tem
  // de fechar depois dela. O ref barra o segundo clique antes de o estado
  // chegar à tela; o servidor também o barra, e o segundo sairia 200 sem
  // efeito.
  const [cancelamentoAberto, setCancelamentoAberto] = useState(false);
  const cancelamentoEmVoo = useRef(false);
  const [cancelando, setCancelando] = useState(false);
  const [erroDoCancelamento, setErroDoCancelamento] = useState<string | null>(null);
  // A frase da corrida perdida, no lugar da de `porQueNaoCancela`. Não
  // envelhece: o Pedido que saiu da janela não volta a ela.
  const [avisoDaCorrida, setAvisoDaCorrida] = useState<string | null>(null);
  // Ao fechar depois de uma releitura, o foco vai ao título, e não ao botão
  // que abriu o Dialog — ele já não existe.
  const focarTituloAoFechar = useRef(false);

  useEffect(() => {
    let vivo = true;
    let timer: ReturnType<typeof setInterval> | undefined;
    let atual = 0;
    // Só a consulta mais recente escreve na tela: a do intervalo que saiu
    // antes do clique pode voltar depois da releitura forçada, e traria de
    // volta o Pedido recusado — com o botão — por um intervalo inteiro.
    let ultima = 0;

    // Trocar de ritmo é recriar o intervalo: setInterval não muda de período
    // depois de armado. O `atual` evita reprogramar a cada consulta, o que
    // adiaria a próxima para sempre.
    function reprogramar(ms: number) {
      if (atual === ms) return;
      atual = ms;
      clearInterval(timer);
      timer = setInterval(consultar, ms);
    }

    async function consultar() {
      const minha = ++ultima;
      try {
        const resposta = await fetch(`/api/v1/pedidos/${encodeURIComponent(pedidoId)}`, {
          credentials: "same-origin",
        });
        const corpo = await resposta.json().catch(() => null);
        if (!vivo || minha !== ultima) return;
        if (!resposta.ok) {
          // Sessão expirada leva ao Login com o caminho atual no `destino`: a
          // tela não tem o que mostrar sem Sessão, e o Comprador volta a este
          // mesmo Pedido depois de entrar. A consulta para junto.
          if (resposta.status === 401) {
            clearInterval(timer);
            router.push(paraLogin());
            return;
          }
          // A mensagem exibida é a do envelope: é ela que fala a Voice and
          // Tone, e o front não reescreve erro do servidor.
          setErro(corpo?.erro?.mensagem ?? "Não foi possível ler o Pedido.");
          // Parar só no que é definitivo: sem Pedido, consultar de novo dá a
          // mesma resposta. Um 500 ou 502 no caminho é transitório, e desistir
          // dele congelaria a tela justamente na janela em que ela existe para
          // esperar.
          if (resposta.status === 404) clearInterval(timer);
          return;
        }
        setErro(null);
        setPedido(corpo);
        // O aviso da nova Tentativa é da superfície recusada: com o Pedido
        // andando, ele envelheceria e voltaria numa recusa futura.
        if (corpo?.status !== PAGAMENTO_RECUSADO) setErroDaTentativa(null);
        const proxima = intervaloDaConsulta(corpo);
        if (proxima === null) {
          clearInterval(timer);
          return;
        }
        reprogramar(proxima);
      } catch {
        if (vivo && minha === ultima) setErro("Não foi possível ler o Pedido.");
      }
    }

    consultarAgora.current = consultar;
    consultar();
    reprogramar(INTERVALO_AGUARDANDO_MS);
    return () => {
      vivo = false;
      clearInterval(timer);
    };
  }, [pedidoId, router]);

  // "Tentar pagar de novo": o Go decide tudo — nova Reserva, teto, corrida —,
  // e a tela executa o desfecho de `desfechoDaNovaTentativa`. No sucesso e nas
  // recusas em que o Pedido mudou, relê: é a leitura que traz o relógio de
  // volta com o prazo novo, ou que tira o botão e diz por quê.
  async function tentarDeNovo() {
    if (emVoo.current) return;
    emVoo.current = true;
    setEnviando(true);
    setErroDaTentativa(null);
    try {
      const resposta = await fetch(rotaDaNovaTentativa(pedidoId), {
        method: "POST",
        credentials: "same-origin",
      });
      const json = await resposta.json().catch(() => null);
      const desfecho = desfechoDaNovaTentativa({ resposta, json });
      if (desfecho.tipo === "semSessao") {
        router.push(paraLogin());
        return;
      }
      if (desfecho.tipo === "erro") {
        setErroDaTentativa(desfecho.mensagem);
        return;
      }
      await consultarAgora.current();
      if (desfecho.aviso) {
        setErroDaTentativa(desfecho.aviso);
      } else if (resposta.ok) {
        titulo.current?.focus();
      }
    } catch {
      setErroDaTentativa(FALHA_NA_NOVA_TENTATIVA);
    } finally {
      emVoo.current = false;
      setEnviando(false);
    }
  }

  // "Cancelar Pedido", confirmado no Dialog: o Go decide tudo — janela,
  // Reserva, corrida —, e a tela executa `desfechoDoCancelamento`. No sucesso
  // e na corrida perdida, fecha e relê: é a leitura que traz o Status novo e a
  // linha do tempo. Erro que não é corrida fica no Dialog, que continua aberto.
  async function cancelar() {
    if (cancelamentoEmVoo.current) return;
    cancelamentoEmVoo.current = true;
    setCancelando(true);
    setErroDoCancelamento(null);
    // Durante o envio o Dialog não fecha: uma requisição pendurada o prenderia
    // aberto para sempre. Esgotado o prazo, o `fetch` aborta, o `catch` mostra
    // a falha no Dialog e o `finally` solta a guarda. Tentar de novo é seguro —
    // sobre o Pedido já cancelado, a rota responde 200 sem efeito.
    const abortar = new AbortController();
    const prazo = setTimeout(() => abortar.abort(), PRAZO_DO_CANCELAMENTO_MS);
    try {
      const resposta = await fetch(rotaDoCancelamento(pedidoId), {
        method: "POST",
        credentials: "same-origin",
        signal: abortar.signal,
      });
      const json = await resposta.json().catch(() => null);
      clearTimeout(prazo);
      const desfecho = desfechoDoCancelamento({ resposta, json });
      if (desfecho.tipo === "semSessao") {
        router.push(paraLogin());
        return;
      }
      if (desfecho.tipo === "erro") {
        setErroDoCancelamento(desfecho.mensagem);
        return;
      }
      // Fecha antes de reler: com o Dialog aberto, o resto da página está
      // escondido do leitor de tela, e a região viva do Status mudaria sem
      // ser anunciada. O foco vai ao título, e não ao botão, que some.
      focarTituloAoFechar.current = true;
      setCancelamentoAberto(false);
      if (desfecho.aviso) setAvisoDaCorrida(desfecho.aviso);
      await consultarAgora.current();
    } catch {
      setErroDoCancelamento(FALHA_NO_CANCELAMENTO);
    } finally {
      clearTimeout(prazo);
      cancelamentoEmVoo.current = false;
      setCancelando(false);
    }
  }

  const superficie = pedido ? superficieDoPedido(pedido) : null;
  const textosDoDialog = textosDoCancelamento(pedido ? pedido.numero : "");
  const semCancelamento = pedido ? fraseSemCancelamento(pedido, avisoDaCorrida) : null;

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="space-y-4">
          <h1 ref={titulo} tabIndex={-1} className="text-2xl font-medium outline-none">Pedido {pedido ? pedido.numero : "…"}</h1>
          {/* A região vive desde o primeiro render e só o conteúdo muda: uma
              região `role="status"` que entra no DOM junto com o texto costuma
              não ser anunciada por leitor de tela — e é exatamente esta
              mudança que o Comprador está esperando ouvir. */}
          <p className="text-sm" role="status">
            {pedido ? (
              <>
                Status: <SeloDoStatus status={pedido.status} />
                {/* O motivo mora na região viva: é ele que o Comprador
                    precisa ouvir quando a tela troca o relógio. */}
                {superficie === "recusado" && (
                  <span className="block font-medium">{motivoDaRecusa(pedido.historico)}</span>
                )}
                {/* Por que não há "Cancelar Pedido" — ou, depois de uma
                    corrida perdida, o que aconteceu. Na região viva porque
                    entra junto do Status novo, e é ele que o explica. */}
                {semCancelamento && (
                  <span className="text-muted-foreground block">{semCancelamento}</span>
                )}
              </>
            ) : (
              !erro && "Carregando o Pedido…"
            )}
          </p>

          {pedido && superficie === "processando" && (
            <>
              <p className="text-sm">
                A confirmação do pagamento chega sozinha, e você não precisa ficar nesta tela.
              </p>
              {pedido.expira_em && <Relogio key={pedido.expira_em} expiraEm={pedido.expira_em} />}
              <Preco centavos={pedido.total_centavos} />
            </>
          )}

          {pedido && superficie === "recusado" && (
            <>
              <p className="text-sm">{textoDasTentativas(pedido.tentativas_restantes)}</p>
              {/* Quando a ação sai da tela pelo Estoque, a razão fica no lugar
                  dela, nomeando o Produto (FR-27). "Cancelar Pedido" continua
                  logo abaixo: cancelar é a outra saída. */}
              {impedimentosDaNovaTentativa(pedido).map(({ chave, texto }) => (
                <p key={chave} className="text-sm">
                  {texto}
                </p>
              ))}
              <p className="text-muted-foreground text-sm">
                Os Itens de Pedido continuam no próprio Pedido.
              </p>
              {podeTentarDeNovo(pedido) && (
                // Primário do shadcn, sem pill nem laranja: não é botão de ação
                // da loja (UX-DR5), e o laranja é só do Confirmar Pedido
                // (UX-DR7). `aria-disabled`, e não `disabled`, para o foco não
                // cair no vazio durante o envio.
                <Button
                  type="button"
                  onClick={tentarDeNovo}
                  aria-disabled={enviando}
                  className="aria-disabled:cursor-not-allowed aria-disabled:opacity-50"
                >
                  {enviando ? "Iniciando a nova Tentativa…" : "Tentar pagar de novo"}
                </Button>
              )}
              {erroDaTentativa && (
                <p className="text-destructive text-sm" role="alert">
                  {erroDaTentativa}
                </p>
              )}
            </>
          )}

          {/* O cancelamento nas três superfícies: o botão nos quatro Status
              da janela; fora dela, a frase que explica por que ele não está
              mora na região viva do Status, logo acima. Botão
              `outline`, sem laranja nem verde (UX-DR7): o destrutivo é o do
              Dialog, onde o passo irreversível acontece. */}
          {pedido && podeCancelar(pedido) && (
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setErroDoCancelamento(null);
                setCancelamentoAberto(true);
              }}
            >
              {textosDoDialog.botao}
            </Button>
          )}
          {pedido && (
            <p className="text-muted-foreground text-sm">
              Última atualização: <Instante valor={pedido.atualizado_em} />
            </p>
          )}

          {erro && (
            <p className="text-destructive text-sm" role="alert">
              {erro}
            </p>
          )}

          <Button asChild variant="outline">
            <a href="/pedidos">Ver o Pedido em Meus pedidos</a>
          </Button>
        </CardContent>
      </Card>

      {/* Recusado também mostra o que o Pedido contém (EXPERIENCE, "Pagamento
          recusado" e "Tentativa expirada" são estados do Detalhe): o
          Comprador vê o que continua no Pedido. Só o relógio o esconde. */}
      {pedido && superficie !== "processando" && <Detalhe pedido={pedido} />}

      {/* Durante o envio o Dialog não fecha — nem por Esc, nem por clique
          fora, nem pelo "Manter o Pedido" —, e os dois botões ficam na tela,
          com `aria-disabled` em vez de `disabled`, para o foco não cair no
          vazio (UX-DR11). Sem o X do canto: "Manter o Pedido" é a saída. */}
      <Dialog
        open={cancelamentoAberto}
        onOpenChange={(aberto) => {
          if (!aberto && !cancelamentoEmVoo.current) setCancelamentoAberto(false);
        }}
      >
        <DialogContent
          showCloseButton={false}
          onCloseAutoFocus={(e) => {
            if (!focarTituloAoFechar.current) return;
            focarTituloAoFechar.current = false;
            e.preventDefault();
            titulo.current?.focus();
          }}
        >
          <DialogHeader>
            <DialogTitle>{textosDoDialog.titulo}</DialogTitle>
            <DialogDescription>{textosDoDialog.descricao}</DialogDescription>
          </DialogHeader>
          {erroDoCancelamento && (
            <p className="text-destructive text-sm" role="alert">
              {erroDoCancelamento}
            </p>
          )}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              aria-disabled={cancelando}
              className="aria-disabled:cursor-not-allowed aria-disabled:opacity-50"
              onClick={() => {
                if (!cancelamentoEmVoo.current) setCancelamentoAberto(false);
              }}
            >
              {textosDoDialog.manter}
            </Button>
            <Button
              type="button"
              variant="destructive"
              aria-disabled={cancelando}
              // O `destructive` do shadcn tinge a 10% e dá 3,99:1 (E3): cheio
              // com branco é o par da 6.7, 4,76:1. O hover escurece — clarear
              // o vermelho baixaria o contraste do branco.
              className="bg-destructive text-white hover:bg-[color-mix(in_oklch,var(--destructive),black_15%)] aria-disabled:hover:bg-destructive aria-disabled:cursor-not-allowed aria-disabled:opacity-50"
              onClick={cancelar}
            >
              {cancelando ? textosDoDialog.enviando : textosDoDialog.confirmar}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
