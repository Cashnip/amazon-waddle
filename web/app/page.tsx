import { Casca } from "@/components/casca";
import { CartaoDeProduto, GRADE, type ProdutoDaVitrine } from "@/components/cartao-de-produto";
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { CatalogoVazio } from "./catalogo-vazio";

// No servidor do Next, `/api/...` não existe: o destino é o do next.config.ts.
const api = (process.env.AZAMON_API_URL || "http://azamon:8080").replace(/\/+$/, "");

type Listagem = { itens: ProdutoDaVitrine[]; pagina: number; por_pagina: number; total: number };

// A Vitrine. A URL é o estado (`?pagina=`): recarregar e voltar no navegador
// reproduzem a página. O carregamento é o `loading.tsx`.
export default async function Vitrine({
  searchParams,
}: {
  searchParams: Promise<{ pagina?: string | string[] }>;
}) {
  const { pagina: texto } = await searchParams;
  const n = Number(texto);
  const pagina = Number.isSafeInteger(n) && n >= 1 ? n : 1;
  // `no-store`: Produto criado ou desativado aparece ou some no próximo acesso.
  const resposta = await fetch(`${api}/api/v1/produtos?pagina=${pagina}`, { cache: "no-store" });
  if (!resposta.ok) throw new Error(`GET /api/v1/produtos = ${resposta.status}`);
  const listagem: Listagem = await resposta.json();
  const paginas = Math.ceil(listagem.total / listagem.por_pagina);

  return (
    <Casca>
      <main className="mx-auto max-w-conteudo space-y-section-sm px-page-margin py-section-sm lg:px-page-margin-lg">
        <h1 className="sr-only">Vitrine</h1>
        {listagem.total === 0 ? (
          <CatalogoVazio />
        ) : listagem.itens.length === 0 ? (
          <p className="py-section-sm text-center">
            Esta página não tem Produtos.{" "}
            <a className="text-link underline" href="/">
              Voltar à primeira página
            </a>
          </p>
        ) : (
          <div className={GRADE}>
            {listagem.itens.map((p) => (
              <CartaoDeProduto key={p.id} produto={p} />
            ))}
          </div>
        )}
        {/* Além do total não há vizinha: o caminho é o link à primeira página. */}
        {paginas > 1 && pagina <= paginas && <Paginacao pagina={pagina} paginas={paginas} />}
      </main>
    </Casca>
  );
}

// A primeira, a última e as vizinhas da atual; o resto vira reticências.
function Paginacao({ pagina, paginas }: { pagina: number; paginas: number }) {
  const numeros = [...new Set([1, pagina - 1, pagina, pagina + 1, paginas])]
    .filter((p) => p >= 1 && p <= paginas)
    .sort((a, b) => a - b);
  return (
    <Pagination aria-label="Paginação">
      <PaginationContent>
        {pagina > 1 && (
          <PaginationItem>
            <PaginationPrevious href={`?pagina=${pagina - 1}`} text="Anterior" aria-label="Página anterior" />
          </PaginationItem>
        )}
        {numeros.map((p, i) => [
          i > 0 && p - numeros[i - 1] > 1 && (
            <PaginationItem key={`r${p}`}>
              <PaginationEllipsis />
            </PaginationItem>
          ),
          <PaginationItem key={p}>
            <PaginationLink href={`?pagina=${p}`} isActive={p === pagina} aria-label={`Página ${p}`}>
              {p}
            </PaginationLink>
          </PaginationItem>,
        ])}
        {pagina < paginas && (
          <PaginationItem>
            <PaginationNext href={`?pagina=${pagina + 1}`} text="Próxima" aria-label="Próxima página" />
          </PaginationItem>
        )}
      </PaginationContent>
    </Pagination>
  );
}
