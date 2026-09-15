"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Card, CardContent } from "@/components/ui/card";
import { paraLogin } from "@/lib/destino";

// A consulta é de 3 s enquanto o Pedido aguarda pagamento — a janela em que a
// confirmação chega — e afrouxa para 10 s no resto do caminho, que anda por
// etapas mais longas. Sem WebSocket (AD-10). Quem diz quando parar é o
// `terminal` do servidor, e não uma lista de Status repetida aqui.
const intervaloAguardando = 3000;
const intervaloAvancando = 10000;

type Pedido = {
  id: string;
  numero: string;
  status: string;
  total_centavos: number;
  atualizado_em: string;
  terminal: boolean;
};

const aguardando = "AGUARDANDO_PAGAMENTO";

// Os rótulos são os termos do glossário, escritos como se lê em tela. Os sete
// do CHECK estão aqui inteiros, embora recusado e cancelado só passem a ser
// alcançáveis nas Épicas 5 e 6: é a lista do banco, e não a do que já acontece.
const rotulo: Record<string, string> = {
  AGUARDANDO_PAGAMENTO: "Aguardando pagamento",
  PAGAMENTO_RECUSADO: "Pagamento recusado",
  PAGO: "Pago",
  EM_SEPARACAO: "Em separação",
  ENVIADO: "Enviado",
  ENTREGUE: "Entregue",
  CANCELADO: "Cancelado",
};

// O instante vem absoluto e em RFC 3339 do servidor; o `dateTime` guarda essa
// forma e o que o Comprador lê é a mesma marca em pt-BR. Não há contagem de
// duração no navegador — só formatação do que o servidor mandou.
function Instante({ valor }: { valor: string }) {
  const data = new Date(valor);
  const texto = Number.isNaN(data.getTime())
    ? valor
    : data.toLocaleString("pt-BR", { dateStyle: "short", timeStyle: "medium" });
  return <time dateTime={valor}>{texto}</time>;
}

function Preco({ centavos }: { centavos: number }) {
  const reais = Math.floor(centavos / 100);
  const resto = String(centavos % 100).padStart(2, "0");
  return (
    <p>
      <span className="preco">R$ {reais.toLocaleString("pt-BR")}</span>
      <span className="preco-centavos align-super">{resto}</span>
    </p>
  );
}

export function Acompanhamento({ pedidoId }: { pedidoId: string }) {
  const router = useRouter();
  const [pedido, setPedido] = useState<Pedido | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  useEffect(() => {
    let vivo = true;
    let timer: ReturnType<typeof setInterval> | undefined;
    let atual = 0;

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
      try {
        const resposta = await fetch(`/api/v1/pedidos/${encodeURIComponent(pedidoId)}`, {
          credentials: "same-origin",
        });
        const corpo = await resposta.json().catch(() => null);
        if (!vivo) return;
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
        // Parar só no estado terminal: a tela existe para acompanhar o Pedido
        // até ENTREGUE, e quem declara o que é terminal é o servidor (AD-18).
        if (corpo?.terminal) {
          clearInterval(timer);
          return;
        }
        reprogramar(corpo?.status === aguardando ? intervaloAguardando : intervaloAvancando);
      } catch {
        if (vivo) setErro("Não foi possível ler o Pedido.");
      }
    }

    consultar();
    reprogramar(intervaloAguardando);
    return () => {
      vivo = false;
      clearInterval(timer);
    };
  }, [pedidoId, router]);

  // Enquanto não há Pedido lido, o erro é tudo o que a tela tem a mostrar.
  // Com Pedido em mão, ele entra ao lado — a consulta continua, e sumir com o
  // Pedido a cada tropeço de rede seria pior que o tropeço.
  if (erro && !pedido) {
    return (
      <p className="text-destructive text-sm" role="alert">
        {erro}
      </p>
    );
  }

  return (
    <Card>
      <CardContent className="space-y-4">
        <h1 className="text-2xl font-medium">
          Pedido {pedido ? pedido.numero : "…"}
        </h1>
        {/* A região vive desde o primeiro render e só o conteúdo muda: uma
            região `role="status"` que entra no DOM junto com o texto costuma
            não ser anunciada por leitor de tela — e é exatamente esta mudança
            que o Comprador está esperando ouvir. */}
        <p className="text-sm" role="status">
          {pedido && (
            <>
              Status: <span className="font-medium">{rotulo[pedido.status] ?? pedido.status}</span>
              {pedido.status === aguardando && " — a confirmação do pagamento chega sozinha."}
            </>
          )}
        </p>
        {pedido && (
          <>
            <Preco centavos={pedido.total_centavos} />
            <p className="text-muted-foreground text-sm">
              Última atualização: <Instante valor={pedido.atualizado_em} />
            </p>
          </>
        )}
        {erro && (
          <p className="text-destructive text-sm" role="alert">
            {erro}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
