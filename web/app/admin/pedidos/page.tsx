import { Suspense } from "react";
import { CascaAdmin } from "@/components/casca-admin";
import { Pedidos } from "./pedidos";

// A Tabela de Pedidos do Administrador (6.4). Como as outras telas
// administrativas, o Server Component não busca nada: a lista é autenticada, e
// o cookie só existe no navegador.
export default function PaginaDePedidosAdmin() {
  return (
    <CascaAdmin>
      <main className="mx-auto max-w-conteudo px-page-margin py-section-sm lg:px-page-margin-lg">
        {/* O filtro inteiro vive na URL: useSearchParams pede o Suspense. */}
        <Suspense>
          <Pedidos />
        </Suspense>
      </main>
    </CascaAdmin>
  );
}
