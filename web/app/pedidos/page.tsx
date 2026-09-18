import { Casca } from "@/components/casca";
import { MeusPedidos } from "./meus-pedidos";

// "Meus pedidos" (2.6) — o esboço que a Estória 6.1 substitui. Molde de
// enderecos/page.tsx: o Server Component só monta a Casca, e o filho
// `"use client"` fala com o Go (AD-10).
export default function PaginaDePedidos() {
  return (
    <Casca>
      <main className="mx-auto max-w-2xl px-page-margin py-section-sm lg:px-page-margin-lg">
        <MeusPedidos />
      </main>
    </Casca>
  );
}
