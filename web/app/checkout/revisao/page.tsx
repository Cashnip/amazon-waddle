import { Casca } from "@/components/casca";
import { EsbocoDaRevisao } from "./esboco-da-revisao";

// O passo Revisão do checkout. Esboço da 5.2, que só mostra o Endereço
// escolhido: a Estória 5.4 substitui esta página pela Revisão de verdade, com
// Frete, total e Confirmar Pedido. Como o passo Endereço, o Server Component
// não busca nada (AD-10).
export default function PaginaDaRevisao() {
  return (
    <Casca>
      <main className="mx-auto max-w-2xl px-page-margin py-section-sm lg:px-page-margin-lg">
        <EsbocoDaRevisao />
      </main>
    </Casca>
  );
}
