"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";

// No molde da Saudacao: quem fala com o Go por caminho relativo é o filho
// `"use client"`, e é essa ida e volta que faz o cookie de Sessão atravessar o
// rewrites(). Nenhuma rota nova no Next e nenhuma regra aqui — o Node é casca.
//
// A tela de acompanhamento do Pedido é da 1.7; aqui o Comprador só vê o
// Pedido nascer, na própria Página de Produto.
type PedidoNascido = { numero: string };

export function Comprar({ produtoId }: { produtoId: string }) {
  const [pedido, setPedido] = useState<PedidoNascido | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [enviando, setEnviando] = useState(false);

  async function confirmar() {
    setEnviando(true);
    setErro(null);
    try {
      const resposta = await fetch("/api/v1/pedidos", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ produto_id: produtoId }),
      });
      const corpo = await resposta.json().catch(() => null);
      // Sem `numero` não há o que confirmar, mesmo com 201: cair calado aqui
      // deixaria a tela sem confirmação e sem erro, e o Comprador compraria
      // de novo. A mensagem exibida é sempre a do envelope — é ela que fala a
      // Voice and Tone, e o front não reescreve erro do servidor.
      if (!resposta.ok || !corpo?.numero) {
        setErro(corpo?.erro?.mensagem ?? "Não foi possível confirmar a compra.");
        return;
      }
      setPedido(corpo);
    } catch {
      setErro("Não foi possível confirmar a compra.");
    } finally {
      setEnviando(false);
    }
  }

  return (
    <div className="space-y-2">
      {/* primary-strong é o laranja reservado ao passo irreversível, e aparece
          uma vez por fluxo (DESIGN.md). Confirmar a compra é esse passo. */}
      {!pedido && (
        <Button
          size="lg"
          onClick={confirmar}
          disabled={enviando}
          className="bg-primary-strong text-primary-strong-foreground hover:bg-primary-strong/90"
        >
          {enviando ? "Confirmando…" : "Confirmar compra"}
        </Button>
      )}
      {/* A região vive desde o primeiro render e só o conteúdo muda: uma
          região `role="status"` que entra no DOM junto com o texto costuma
          não ser anunciada por leitor de tela. */}
      <p className="text-sm" role="status">
        {pedido && (
          <>
            Pedido <span className="font-medium">{pedido.numero}</span> criado. Aguardando
            pagamento.
          </>
        )}
      </p>
      {erro && (
        <p className="text-destructive text-sm" role="alert">
          {erro}
        </p>
      )}
    </div>
  );
}
