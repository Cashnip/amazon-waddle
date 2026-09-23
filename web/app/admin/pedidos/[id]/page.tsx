import { CascaAdmin } from "@/components/casca-admin";
import { DetalheDoPedidoAdmin } from "./detalhe-admin";

// O Detalhe administrativo de um Pedido (6.4). Como as outras telas
// administrativas, o Server Component não busca nada: a leitura é autenticada,
// e o cookie de Sessão só existe no navegador.
export default async function PaginaDoPedidoAdmin({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;

  return (
    <CascaAdmin>
      <main className="mx-auto max-w-conteudo px-page-margin py-section-sm lg:px-page-margin-lg">
        <DetalheDoPedidoAdmin pedidoId={id} />
      </main>
    </CascaAdmin>
  );
}
