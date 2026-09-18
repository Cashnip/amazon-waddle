"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Card, CardContent } from "@/components/ui/card";
import { paraLogin } from "@/lib/destino";

// "Meus pedidos" (2.6) — a listagem mínima que a Estória 6.1 substitui: sem
// filtro por Status, sem paginação e sem Skeleton (UX-DR20b) ficam para ela.
// O servidor já filtra por dono na própria consulta (AD-11) e ordena por
// `id DESC` — o mais recente no topo —, então a tela não reordena nada.
type Pedido = { id: string; numero: string; status: string; total_centavos: number };

// Os mesmos rótulos fechados de `pedidos/[id]/acompanhamento.tsx` — copiados,
// e não extraídos para um módulo comum: este arquivo inteiro é o esboço que a
// 6.1 substitui, e uma dependência cruzada com a tela de acompanhamento só
// sobreviveria até lá.
const rotulo: Record<string, string> = {
  AGUARDANDO_PAGAMENTO: "Aguardando pagamento",
  PAGAMENTO_RECUSADO: "Pagamento recusado",
  PAGO: "Pago",
  EM_SEPARACAO: "Em separação",
  ENVIADO: "Enviado",
  ENTREGUE: "Entregue",
  CANCELADO: "Cancelado",
};

// O mesmo formato de preço de `pedidos/[id]/acompanhamento.tsx` — copiado, e
// não importado, pelo mesmo motivo do mapa de rótulos acima. Sai num `<span>`
// aqui, e não num `<p>`: a linha inteira é um `<a>` só, e um `<p>` quebraria o
// fluxo das outras duas colunas dentro dele.
function Preco({ centavos }: { centavos: number }) {
  const reais = Math.floor(centavos / 100);
  const resto = String(centavos % 100).padStart(2, "0");
  return (
    <span>
      <span className="preco">R$ {reais.toLocaleString("pt-BR")}</span>
      <span className="preco-centavos align-super">{resto}</span>
    </span>
  );
}

export function MeusPedidos() {
  const router = useRouter();
  const [pedidos, setPedidos] = useState<Pedido[] | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  // Sessão expirada leva ao Login com o caminho atual no `destino`: a tela
  // não tem o que mostrar sem Sessão, e o Comprador volta a esta mesma lista
  // depois de entrar.
  const carregar = useCallback(async () => {
    try {
      const resposta = await fetch("/api/v1/pedidos", { credentials: "same-origin" });
      if (resposta.status === 401) {
        router.push(paraLogin());
        return;
      }
      const corpo = await resposta.json().catch(() => null);
      if (!resposta.ok) {
        setErro(corpo?.erro?.mensagem ?? "Não foi possível ler os Pedidos.");
        return;
      }
      setErro(null);
      setPedidos(corpo ?? []);
    } catch {
      setErro("Não foi possível falar com o servidor.");
    }
  }, [router]);

  useEffect(() => {
    carregar();
  }, [carregar]);

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-medium">Meus pedidos</h1>

      {erro && (
        <Alert variant="destructive">
          <AlertDescription>{erro}</AlertDescription>
        </Alert>
      )}

      {/* Antes da primeira leitura não há lista nem vazio: dizer "Nenhum
          Pedido ainda." enquanto a resposta está a caminho seria uma
          afirmação falsa sobre a conta. */}
      {pedidos?.length === 0 && (
        <Card>
          <CardContent className="text-center">
            <p>Nenhum Pedido ainda.</p>
          </CardContent>
        </Card>
      )}

      {pedidos?.map((pedido) => (
        <Card key={pedido.id}>
          <CardContent>
            <a
              className="text-link flex items-center justify-between gap-4 underline underline-offset-2"
              href={`/pedidos/${pedido.id}`}
            >
              <span>{pedido.numero}</span>
              <span>{rotulo[pedido.status] ?? pedido.status}</span>
              <Preco centavos={pedido.total_centavos} />
            </a>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
