"use client";

import { useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { FALHA_DE_REDE, pedir } from "@/lib/pedir";
import {
  desfechoDaTransicao,
  rotaDaTransicao,
  rotuloDaAcao,
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
}: {
  pedido: Pick<PedidoNaTabela, "id" | "status" | "permitidas">;
  // Relê a superfície: é a leitura que traz o Status novo, o `permitidas` novo
  // e a linha do histórico. Nenhuma tela monta o Pedido novo a partir do que
  // ela já tinha.
  aoMudar: () => Promise<void> | void;
  aoRelatar: (relato: RelatoDaTransicao) => void;
}) {
  const router = useRouter();
  // Um envio por vez: o ref barra o segundo clique antes de o estado chegar à
  // tela. O servidor também o barra — o segundo sai ESTADO_JA_AVANCADO —, mas
  // a tela não precisa gastar a ida.
  const emVoo = useRef(false);
  const [enviando, setEnviando] = useState(false);

  async function acionar(destino: string) {
    if (emVoo.current) return;
    emVoo.current = true;
    setEnviando(true);
    // O desfecho anterior sai agora: mantê-lo ao lado do novo faria a tela
    // explicar duas tentativas ao mesmo tempo.
    aoRelatar({ alerta: null, erro: null });
    try {
      const { resposta, json } = await pedir(rotaDaTransicao(pedido.id), "POST", {
        de: pedido.status,
        para: destino,
      });
      const desfecho = desfechoDaTransicao({ resposta, json }, { de: pedido.status, para: destino });
      if (desfecho.tipo === "semSessao") {
        router.replace("/admin/entrar");
        return;
      }
      if (desfecho.tipo === "erro") {
        aoRelatar({ alerta: null, erro: desfecho.mensagem });
        return;
      }
      // Nas três releituras — sucesso e as duas recusas — o Pedido não é mais
      // o que a tela mostrava, e quem traz o Status novo é a leitura. O relato
      // vai ANTES da releitura: ela pode desmontar este componente.
      aoRelatar({ alerta: desfecho.alerta, erro: null });
      await aoMudar();
    } catch {
      aoRelatar({ alerta: null, erro: FALHA_DE_REDE });
    } finally {
      emVoo.current = false;
      setEnviando(false);
    }
  }

  return (
    <div className="flex flex-wrap justify-end gap-2">
      {pedido.permitidas.map((destino) => (
        // `aria-disabled`, e não `disabled`: durante o envio o botão permanece
        // na tela e no caminho do foco (UX-DR11).
        <Button
          key={destino}
          type="button"
          variant="outline"
          size="sm"
          aria-disabled={enviando}
          className="aria-disabled:cursor-not-allowed aria-disabled:opacity-50"
          onClick={() => acionar(destino)}
        >
          {enviando ? "Salvando…" : rotuloDaAcao(destino)}
        </Button>
      ))}
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
