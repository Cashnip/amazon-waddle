import { Casca } from "@/components/casca";
import { MeusEnderecos } from "./meus-enderecos";

// "Meus endereços" (FR-5). Como a tela do Pedido, o Server Component não busca
// nada: a listagem é autenticada, e o cookie de Sessão só existe no navegador —
// é o filho `"use client"` que fala com o Go por caminho relativo, e é essa ida
// e volta que faz o cookie atravessar o rewrites(). Buscar aqui exigiria
// reencaminhar o cabeçalho Cookie à mão, que é regra no Node (AD-10).
//
// O Menu da conta (2.6) linka para cá — não é mais preciso digitar a URL.
export default function PaginaDeEnderecos() {
  return (
    <Casca>
      <main className="mx-auto max-w-2xl px-page-margin py-section-sm lg:px-page-margin-lg">
        <MeusEnderecos />
      </main>
    </Casca>
  );
}
