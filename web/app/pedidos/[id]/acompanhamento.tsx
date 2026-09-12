"use client";

import { useEffect, useState } from "react";
import { Card, CardContent } from "@/components/ui/card";

// A consulta é de 3 s enquanto o Pedido aguarda pagamento, e para assim que o
// Status avança — sem WebSocket (AD-10). Todo instante exibido é absoluto,
// RFC 3339 e vindo do servidor: o navegador nunca conta duração.
const intervalo = 3000;

type Pedido = {
  id: string;
  numero: string;
  status: string;
  total_centavos: number;
  atualizado_em: string;
};

const aguardando = "AGUARDANDO_PAGAMENTO";

// Os rótulos são os termos do glossário, escritos como se lê em tela.
const rotulo: Record<string, string> = {
  AGUARDANDO_PAGAMENTO: "Aguardando pagamento",
  PAGO: "Pago",
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
  const [pedido, setPedido] = useState<Pedido | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  useEffect(() => {
    let vivo = true;
    let timer: ReturnType<typeof setInterval> | undefined;

    async function consultar() {
      try {
        const resposta = await fetch(`/api/v1/pedidos/${encodeURIComponent(pedidoId)}`, {
          credentials: "same-origin",
        });
        const corpo = await resposta.json().catch(() => null);
        if (!vivo) return;
        if (!resposta.ok) {
          // A mensagem exibida é a do envelope: é ela que fala a Voice and
          // Tone, e o front não reescreve erro do servidor.
          setErro(corpo?.erro?.mensagem ?? "Não foi possível ler o Pedido.");
          // Parar só no que é definitivo: sem Sessão ou sem Pedido, consultar
          // de novo dá a mesma resposta. Um 500 ou 502 no caminho é
          // transitório, e desistir dele congelaria a tela justamente na
          // janela em que ela existe para esperar.
          if (resposta.status === 401 || resposta.status === 404) clearInterval(timer);
          return;
        }
        setErro(null);
        setPedido(corpo);
        // Parar de consultar assim que o Pedido sai de AGUARDANDO_PAGAMENTO é
        // metade do ponto: a tela existe para esperar esse avanço.
        if (corpo?.status !== aguardando) clearInterval(timer);
      } catch {
        if (vivo) setErro("Não foi possível ler o Pedido.");
      }
    }

    consultar();
    timer = setInterval(consultar, intervalo);
    return () => {
      vivo = false;
      clearInterval(timer);
    };
  }, [pedidoId]);

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
