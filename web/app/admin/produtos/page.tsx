import { Suspense } from "react";
import { CascaAdmin } from "@/components/casca-admin";
import { Produtos } from "./produtos";

// A tela Produtos (3.3). Como "Meus endereços", o Server Component não busca
// nada: a lista é autenticada, e o cookie só existe no navegador.
export default function PaginaDeProdutos() {
  return (
    <CascaAdmin>
      <main className="mx-auto max-w-conteudo px-page-margin py-section-sm lg:px-page-margin-lg">
        {/* A página vive na URL: useSearchParams pede o Suspense. */}
        <Suspense>
          <Produtos />
        </Suspense>
      </main>
    </CascaAdmin>
  );
}
