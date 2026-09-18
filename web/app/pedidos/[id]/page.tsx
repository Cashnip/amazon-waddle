import { Casca } from "@/components/casca";
import { Acompanhamento } from "./acompanhamento";

// Tela do Pedido em processamento (FR-25). O Comprador chega aqui depois do
// 201 da compra e vê o Status virar PAGO sem recarregar.
//
// Diferente da Página de Produto, o Server Component não busca nada: a leitura
// do Pedido é autenticada, e o cookie de Sessão só existe no navegador — é o
// filho `"use client"` que fala com o Go por caminho relativo, e é essa ida e
// volta que faz o cookie atravessar o rewrites(). Buscar aqui exigiria
// reencaminhar o cabeçalho Cookie à mão, que é regra no Node — proibido pelo
// AD-10.
export default async function PaginaDePedido({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  return (
    <Casca>
      <main className="mx-auto max-w-conteudo px-page-margin py-section-sm lg:px-page-margin-lg">
        <Acompanhamento pedidoId={id} />
      </main>
    </Casca>
  );
}
