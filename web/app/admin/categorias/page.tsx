import { CascaAdmin } from "@/components/casca-admin";
import { Categorias } from "./categorias";

// A tela Categorias (3.2). Como "Meus endereços", o Server Component não busca
// nada: a lista é autenticada, e o cookie só existe no navegador.
export default function PaginaDeCategorias() {
  return (
    <CascaAdmin>
      <main className="mx-auto max-w-conteudo px-page-margin py-section-sm lg:px-page-margin-lg">
        <Categorias />
      </main>
    </CascaAdmin>
  );
}
