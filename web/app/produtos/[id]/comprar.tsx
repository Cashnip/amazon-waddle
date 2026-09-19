"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { paraLogin } from "@/lib/destino";

// No molde da Saudacao: quem fala com o Go por caminho relativo é o filho
// `"use client"`, e é essa ida e volta que faz o cookie de Sessão atravessar o
// rewrites(). Nenhuma rota nova no Next e nenhuma regra aqui — o Node é casca.
//
// Depois do 201 o Comprador é levado à tela do Pedido em processamento. O `id`
// vem no corpo da criação desde a 1.6, que o incluiu justamente para isto.
export function Comprar({ produtoId }: { produtoId: string }) {
  const router = useRouter();
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
      // Sessão expirada não é erro para ler em linha: o Comprador é levado ao
      // Login com o caminho atual no `destino`, e volta a esta mesma tela
      // depois de entrar. O botão continua desabilitado durante a navegação.
      if (resposta.status === 401) {
        router.push(paraLogin());
        return;
      }
      // Sem `id` não há para onde levar, mesmo com 201: cair calado aqui
      // deixaria a tela sem confirmação e sem erro, e o Comprador compraria
      // de novo. A mensagem exibida é sempre a do envelope — é ela que fala a
      // Voice and Tone, e o front não reescreve erro do servidor.
      if (!resposta.ok || !corpo?.id) {
        setErro(corpo?.erro?.mensagem ?? "Não foi possível confirmar a compra.");
        setEnviando(false);
        return;
      }
      // O botão continua desabilitado durante a navegação: reabilitá-lo abriria
      // a janela para uma segunda compra enquanto a tela ainda é esta.
      router.push(`/pedidos/${corpo.id}`);
    } catch {
      setErro("Não foi possível confirmar a compra.");
      setEnviando(false);
    }
  }

  return (
    <div className="space-y-2">
      {/* primary-strong é o laranja reservado ao passo irreversível, e aparece
          uma vez por fluxo (DESIGN.md). Confirmar a compra é esse passo. */}
      <Button
        size="lg"
        onClick={confirmar}
        disabled={enviando}
        className="rounded-full bg-primary-strong text-primary-strong-foreground hover:bg-primary-strong/90"
      >
        {enviando ? "Confirmando…" : "Confirmar compra"}
      </Button>
      {erro && (
        <p className="text-destructive text-sm" role="alert">
          {erro}
        </p>
      )}
    </div>
  );
}
