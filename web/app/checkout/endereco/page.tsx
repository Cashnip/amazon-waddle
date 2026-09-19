import { Casca } from "@/components/casca";
import { EscolhaDeEndereco } from "./escolha-de-endereco";

// O primeiro dos dois passos do checkout, Endereço → Revisão (FR-20). No molde
// do Carrinho: o Server Component não busca nada, porque os Endereços e o
// Carrinho são autenticados e o cookie de Sessão só existe no navegador — quem
// fala com o Go por caminho relativo é o filho `"use client"` (AD-10).
export default function PaginaDoEnderecoNoCheckout() {
  return (
    <Casca>
      <main className="mx-auto max-w-2xl px-page-margin py-section-sm lg:px-page-margin-lg">
        <EscolhaDeEndereco />
      </main>
    </Casca>
  );
}
