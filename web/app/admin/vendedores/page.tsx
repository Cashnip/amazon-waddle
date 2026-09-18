import { CascaAdmin } from "@/components/casca-admin";
import { Vendedores } from "./vendedores";

// A tela Vendedores (3.1). Como "Meus endereços", o Server Component não busca
// nada: a lista é autenticada, e o cookie só existe no navegador.
export default function PaginaDeVendedores() {
  return (
    <CascaAdmin>
      <main className="mx-auto max-w-conteudo px-page-margin py-section-sm lg:px-page-margin-lg">
        <Vendedores />
      </main>
    </CascaAdmin>
  );
}
