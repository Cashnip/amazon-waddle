"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FALHA_DE_REDE, pedir } from "@/lib/pedir";
import {
  desfechoDaTransicao,
  rotaDaTransicao,
  rotuloDaAcao,
  textosDaTransicao,
  type PedidoNaTabela,
  type RelatoDaTransicao,
} from "@/lib/pedido";

// O bloco de ação do painel do Administrador (6.4, FR-32): os botões das
// transições disponíveis. Mora num componente só porque aparece nas DUAS
// superfícies — a linha da Tabela e o Detalhe —, e dois tratamentos do mesmo
// desfecho divergiriam na primeira mudança.
//
// Os botões saem de `permitidas`, que o Go manda pronto a partir da tabela do
// AD-3 com `AtorAdministrador`: nenhuma transição é declarada aqui (AD-10). É
// por isso que não existe um quarto botão de cancelar — a tabela não tem essa
// linha para este ator, e não há `if` nenhum a esquecer.
//
// O `de` enviado é o Status que ESTA TELA leu, e não o de agora: é ele que faz
// o compare-and-swap do Go distinguir "o Pedido já avançou" de "esta transição
// não existe". Mandar o Status lido no servidor apagaria a corrida com a
// simulação, e a recusa sairia com causa falsa.
//
// O desfecho é REPORTADO para cima, e não renderizado aqui: com a Tabela
// filtrada por Status, a releitura tira o Pedido da lista e este componente
// desmonta — o Alert sumiria junto, e o Administrador não leria nada
// justamente no caso que a estória existe para nomear.
export function AcoesDoPedido({
  pedido,
  aoMudar,
  aoRelatar,
  aoConfirmar,
}: {
  pedido: Pick<PedidoNaTabela, "id" | "numero" | "status" | "permitidas">;
  // Relê a superfície: é a leitura que traz o Status novo, o `permitidas` novo
  // e a linha do histórico. Nenhuma tela monta o Pedido novo a partir do que
  // ela já tinha.
  aoMudar: () => Promise<void> | void;
  aoRelatar: (relato: RelatoDaTransicao) => void;
  // Avisa a superfície que o Dialog abriu ou fechou: a Tabela pausa a
  // consulta de 10 s, que sob filtro tiraria a linha — e o Dialog junto — no
  // meio da decisão.
  aoConfirmar?: (aberto: boolean) => void;
}) {
  const router = useRouter();
  // O clique só escolhe o destino e abre o Dialog; o POST sai do confirmar
  // (EXPERIENCE: avançar Status é irreversível e pede confirmação).
  // O destino fica guardado depois de fechar: o Dialog anima a saída, e o
  // texto não pode sumir antes dele.
  // O `de` é o Status que a tela mostrava NO CLIQUE: se a consulta o mudar com
  // o Dialog aberto, o compare-and-swap do Go ainda nomeia a corrida.
  const [destino, setDestino] = useState<{ de: string; para: string } | null>(null);
  const [aberto, setAbertoLocal] = useState(false);
  function setAberto(v: boolean) {
    setAbertoLocal(v);
    aoConfirmar?.(v);
  }
  // Um envio por vez: o ref barra o segundo clique antes de o estado chegar à
  // tela. O servidor também o barra — o segundo sai ESTADO_JA_AVANCADO —, mas
  // a tela não precisa gastar a ida.
  const emVoo = useRef(false);
  const [enviando, setEnviando] = useState(false);
  // Depois de um envio, o botão que abriu o Dialog em geral some na releitura
  // (o destino deixa de ser permitido, ou a linha sai do filtro): o foco não
  // volta a ele, e vai ao primeiro botão que o bloco ainda tiver.
  const bloco = useRef<HTMLDivElement>(null);
  const enviou = useRef(false);
  // O foco espera o commit da releitura: logo depois do `await aoMudar()` o
  // DOM ainda tem o botão velho, que some no render seguinte.
  const [releituras, setReleituras] = useState(0);
  useEffect(() => {
    if (releituras > 0) focarBloco();
  }, [releituras]);

  async function confirmar() {
    if (emVoo.current || destino === null || !aberto) return;
    emVoo.current = true;
    setEnviando(true);
    const tentada = { numero: pedido.numero, ...destino };
    // O desfecho anterior sai agora: mantê-lo ao lado do novo faria a tela
    // explicar duas tentativas ao mesmo tempo.
    aoRelatar({ alerta: null, erro: null });
    try {
      const { resposta, json } = await pedir(rotaDaTransicao(pedido.id), "POST", {
        de: tentada.de,
        para: tentada.para,
      });
      const desfecho = desfechoDaTransicao({ resposta, json }, tentada);
      if (desfecho.tipo === "semSessao") {
        setAberto(false);
        router.replace("/admin/entrar");
        return;
      }
      // Fecha antes de relatar: com o Dialog aberto, o resto da página está
      // escondido do leitor de tela, e o Alert entraria sem ser anunciado. No
      // erro o botão continua lá, e o foco volta a ele como o Radix faz.
      setAberto(false);
      if (desfecho.tipo === "erro") {
        aoRelatar({ alerta: null, erro: desfecho.mensagem });
        return;
      }
      enviou.current = true;
      // Nas três releituras — sucesso e as duas recusas — o Pedido não é mais
      // o que a tela mostrava, e quem traz o Status novo é a leitura. O relato
      // vai ANTES da releitura: ela pode desmontar este componente.
      aoRelatar({ alerta: desfecho.alerta, erro: null });
      await aoMudar();
      setReleituras((n) => n + 1);
    } catch {
      setAberto(false);
      aoRelatar({ alerta: null, erro: FALHA_DE_REDE });
    } finally {
      emVoo.current = false;
      setEnviando(false);
    }
  }

  function focarBloco() {
    bloco.current?.querySelector("button")?.focus();
  }

  const textos = destino === null ? null : textosDaTransicao(pedido.numero, destino.para);

  return (
    <div ref={bloco} className="flex flex-wrap justify-end gap-2">
      {pedido.permitidas.map((d) => (
        // `aria-disabled`, e não `disabled`: durante o envio o botão permanece
        // na tela e no caminho do foco (UX-DR11).
        <Button
          key={d}
          type="button"
          variant="outline"
          size="sm"
          aria-disabled={enviando}
          className="aria-disabled:cursor-not-allowed aria-disabled:opacity-50"
          onClick={() => {
            if (emVoo.current) return;
            setDestino({ de: pedido.status, para: d });
            setAberto(true);
          }}
        >
          {rotuloDaAcao(d)}
        </Button>
      ))}

      {/* Durante o envio o Dialog não fecha — nem por Esc, nem por clique
          fora, nem pelo "Voltar" —, e os dois botões ficam na tela com
          `aria-disabled` (o mesmo Dialog do cancelamento, 6.3). O confirmar
          não é laranja: o laranja é do checkout (DESIGN). */}
      <Dialog
        open={aberto}
        onOpenChange={(abrir) => {
          if (!abrir && !emVoo.current) setAberto(false);
        }}
      >
        <DialogContent
          showCloseButton={false}
          onCloseAutoFocus={(e) => {
            if (!enviou.current) return;
            enviou.current = false;
            e.preventDefault();
            focarBloco();
          }}
        >
          {textos && (
            <>
              <DialogHeader>
                <DialogTitle>{textos.titulo}</DialogTitle>
                <DialogDescription>{textos.descricao}</DialogDescription>
              </DialogHeader>
              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  aria-disabled={enviando}
                  className="aria-disabled:cursor-not-allowed aria-disabled:opacity-50"
                  onClick={() => {
                    if (!emVoo.current) setAberto(false);
                  }}
                >
                  {textos.voltar}
                </Button>
                <Button
                  type="button"
                  aria-disabled={enviando}
                  className="aria-disabled:cursor-not-allowed aria-disabled:opacity-50"
                  onClick={confirmar}
                >
                  {enviando ? textos.enviando : textos.confirmar}
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

// O desfecho da transição, renderizado pela superfície e fora da linha. As
// duas regiões existem desde o primeiro render e só o conteúdo muda: uma
// região viva que entra no DOM junto com o texto costuma não ser anunciada por
// leitor de tela — e é exatamente esta mudança que o Administrador espera ouvir.
//
// A separação das variantes é a que a EXPERIENCE pede: a transição fora da
// tabela é defeito de quem chamou e sai `destructive`; o Pedido que outro ator
// já moveu é INFORMAÇÃO, e sai neutro — dizer "transição inválida" ali
// reportaria causa falsa. Nunca `Toast`.
export function DesfechoNaTela({ relato }: { relato: RelatoDaTransicao }) {
  const informativo = relato.alerta?.variante === "informativo" ? relato.alerta.texto : null;
  const destrutivo = relato.alerta?.variante === "destrutivo" ? relato.alerta.texto : relato.erro;
  return (
    <div className="space-y-2">
      <div role="status">
        {informativo && (
          <Alert>
            <AlertDescription>{informativo}</AlertDescription>
          </Alert>
        )}
      </div>
      <div role="alert">
        {destrutivo && (
          <Alert variant="destructive">
            <AlertDescription>{destrutivo}</AlertDescription>
          </Alert>
        )}
      </div>
    </div>
  );
}
