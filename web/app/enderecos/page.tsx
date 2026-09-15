import { Saudacao } from "@/app/produtos/[id]/saudacao";
import { MeusEnderecos } from "./meus-enderecos";

// "Meus endereços" (FR-5). Como a tela do Pedido, o Server Component não busca
// nada: a listagem é autenticada, e o cookie de Sessão só existe no navegador —
// é o filho `"use client"` que fala com o Go por caminho relativo, e é essa ida
// e volta que faz o cookie atravessar o rewrites(). Buscar aqui exigiria
// reencaminhar o cabeçalho Cookie à mão, que é regra no Node (AD-10).
//
// Não há link para cá na casca: o Menu da conta é da 2.6. O endereço da tela é
// `/enderecos`, e é por ele que a demonstração entra.
export default function PaginaDeEnderecos() {
  return (
    <div className="min-h-svh">
      <header className="bg-chrome text-chrome-foreground">
        <div className="mx-auto flex max-w-conteudo items-center justify-between px-page-margin py-4 lg:px-page-margin-lg">
          <a className="wordmark" href="/">
            azamon
          </a>
          <Saudacao />
        </div>
      </header>

      <main className="mx-auto max-w-2xl px-page-margin py-section-sm lg:px-page-margin-lg">
        <MeusEnderecos />
      </main>
    </div>
  );
}
