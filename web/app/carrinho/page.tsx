import { Casca } from "@/components/casca";
import { MeuCarrinho } from "./meu-carrinho";

// O Carrinho do Comprador (FR-16..FR-18). No molde de "Meus endereços": o
// Server Component não busca nada, porque o Carrinho é autenticado e o cookie de
// Sessão só existe no navegador — quem fala com o Go por caminho relativo é o
// filho `"use client"`. Buscar aqui exigiria reencaminhar o cabeçalho Cookie à
// mão, que é regra no Node (AD-10).
export default function PaginaDoCarrinho() {
  return (
    <Casca>
      <main className="mx-auto max-w-3xl px-page-margin py-section-sm lg:px-page-margin-lg">
        <MeuCarrinho />
      </main>
    </Casca>
  );
}
