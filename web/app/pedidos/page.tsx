import { Suspense } from "react";
import { Casca } from "@/components/casca";
import { MeusPedidos } from "./meus-pedidos";

// "Meus pedidos" (6.1). Molde de enderecos/page.tsx: o Server Component só
// monta a Casca, e o filho `"use client"` fala com o Go (AD-10).
//
// O `<Suspense>` é obrigação do Next, e não decoração: a tela lê a página da
// URL por `useSearchParams`, e sem a fronteira o `next build` recusa a rota
// inteira. O fallback fica vazio — quem mostra o Skeleton é a própria tela,
// que monta no mesmo instante.
export default function PaginaDePedidos() {
  return (
    <Casca>
      <main className="mx-auto max-w-2xl px-page-margin py-section-sm lg:px-page-margin-lg">
        <Suspense>
          <MeusPedidos />
        </Suspense>
      </main>
    </Casca>
  );
}
