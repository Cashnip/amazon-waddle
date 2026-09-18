import { Casca } from "@/components/casca";
import { Perfil } from "./perfil";

// Tela de Perfil (2.6). Molde de enderecos/page.tsx: o Server Component só
// monta a Casca — a leitura é autenticada, e o cookie de Sessão só existe no
// navegador, então é o filho `"use client"` quem fala com o Go (AD-10).
export default function PaginaDePerfil() {
  return (
    <Casca>
      <main className="mx-auto max-w-md px-page-margin py-section-sm">
        <Perfil />
      </main>
    </Casca>
  );
}
