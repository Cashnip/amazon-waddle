import { Casca } from "@/components/casca";
import { RevisaoDoPedido } from "./revisao-do-pedido";

// O passo Revisão do checkout (FR-22): Itens com preço unitário e parcela,
// Endereço escolhido, Subtotal/Frete/Total vindos do Go, a declaração de
// pagamento e o Confirmar Pedido. Como o passo Endereço, o Server Component
// não busca nada (AD-10) — a leitura é autenticada e o filho `"use client"` a
// faz por caminho relativo.
export default function PaginaDaRevisao() {
  return (
    <Casca>
      <main className="mx-auto max-w-2xl px-page-margin py-section-sm lg:px-page-margin-lg">
        <RevisaoDoPedido />
      </main>
    </Casca>
  );
}
