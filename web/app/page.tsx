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
import { formatarPreco } from "@/lib/preco";
import { CatalogoVazio } from "./catalogo-vazio";
import { Ordenar, PainelDeFiltros, type Categoria, type Estado } from "./filtros";

// No servidor do Next, `/api/...` não existe: o destino é o do next.config.ts.
const api = (process.env.AZAMON_API_URL || "http://azamon:8080").replace(/\/+$/, "");

type Listagem = { itens: ProdutoDaVitrine[]; pagina: number; por_pagina: number; total: number };

type Parametros = Partial<Record<keyof Estado | "pagina", string | string[]>>;

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const ORDENACOES = ["recentes", "preco_asc", "preco_desc"];

// A URL crua vira um estado que o Go aceita: valor que ele recusaria (termo
// além do teto, preço que não é centavo, ordenação desconhecida, Categoria que
// não é uuid) cai, em vez de virar página de erro.
function estadoDe(p: Parametros): Estado {
  const um = (v?: string | string[]) => (Array.isArray(v) ? v[0] : v) ?? "";
  const centavos = (v: string) => (/^\d{1,15}$/.test(v) ? v : "");
  const ordenacao = um(p.ordenacao);
  const categoria = um(p.categoria);
  return {
    termo: Array.from(um(p.termo).trim()).slice(0, 100).join(""),
    // Minúsculo: é assim que a lista de Categorias o traz, e o chip o procura nela.
    categoria: UUID.test(categoria) ? categoria.toLowerCase() : "",
    preco_min: centavos(um(p.preco_min)),
    preco_max: centavos(um(p.preco_max)),
    ordenacao: ORDENACOES.includes(ordenacao) ? ordenacao : "",
  };
}

// A querystring do estado, com mudanças; vazio sai. Sem `pagina`, é a 1.
function consulta(estado: Estado, muda: Partial<Estado> & { pagina?: number } = {}) {
  const q = new URLSearchParams();
  for (const [chave, valor] of Object.entries({ ...estado, ...muda })) if (valor) q.set(chave, String(valor));
  return q.toString();
}

function link(estado: Estado, muda: Partial<Estado> & { pagina?: number } = {}) {
  const q = consulta(estado, muda);
  return q ? `/?${q}` : "/";
}

// A Vitrine e os Resultados de busca são a mesma tela. A URL é o estado
// (termo, Categoria, faixa, ordenação e página): recarregar e voltar no
// navegador reproduzem a listagem. O carregamento é o `loading.tsx`.
export default async function Vitrine({ searchParams }: { searchParams: Promise<Parametros> }) {
  const parametros = await searchParams;
  const estado = estadoDe(parametros);
  const n = Number(Array.isArray(parametros.pagina) ? parametros.pagina[0] : parametros.pagina);
  const pagina = Number.isSafeInteger(n) && n >= 1 ? n : 1;
  // A faixa invertida não vai ao Go: a lista sai sem o filtro de preço, e o
  // painel mostra a mensagem em linha.
  const faixaInvertida =
    estado.preco_min !== "" && estado.preco_max !== "" && Number(estado.preco_min) > Number(estado.preco_max);
  const pedido = faixaInvertida ? { ...estado, preco_min: "", preco_max: "" } : estado;

  // `no-store`: Produto criado ou desativado aparece ou some no próximo acesso.
  const [resposta, respostaCategorias] = await Promise.all([
    fetch(`${api}/api/v1/produtos?${consulta(pedido, { pagina })}`, { cache: "no-store" }),
    fetch(`${api}/api/v1/categorias`, { cache: "no-store" }),
  ]);
  if (!resposta.ok) throw new Error(`GET /api/v1/produtos = ${resposta.status}`);
  if (!respostaCategorias.ok) throw new Error(`GET /api/v1/categorias = ${respostaCategorias.status}`);
  const listagem: Listagem = await resposta.json();
  const categorias: Categoria[] = await respostaCategorias.json();
  const paginas = Math.ceil(listagem.total / listagem.por_pagina);

  // Os chips: um por filtro presente na URL, cada um um link sem ele.
  const nomeDaCategoria = categorias.find((c) => c.id === estado.categoria)?.nome;
  const chips: { rotulo: string; sem: Partial<Estado> }[] = [];
  if (estado.termo) chips.push({ rotulo: `"${estado.termo}"`, sem: { termo: "" } });
  if (estado.categoria) chips.push({ rotulo: `Categoria: ${nomeDaCategoria ?? "desconhecida"}`, sem: { categoria: "" } });
  if (estado.preco_min) chips.push({ rotulo: `A partir de ${formatarPreco(Number(estado.preco_min))}`, sem: { preco_min: "" } });
  if (estado.preco_max) chips.push({ rotulo: `Até ${formatarPreco(Number(estado.preco_max))}`, sem: { preco_max: "" } });
  const limpar = link({ termo: "", categoria: "", preco_min: "", preco_max: "", ordenacao: estado.ordenacao });

  const inicio = (pagina - 1) * listagem.por_pagina + 1;
  const fim = inicio + listagem.itens.length - 1;
  const contagem =
    listagem.itens.length > 0
      ? `${inicio}–${fim} de ${listagem.total} ${listagem.total === 1 ? "resultado" : "resultados"}`
      : `${listagem.total} ${listagem.total === 1 ? "resultado" : "resultados"}`;

  return (
    <Casca>
      <main className="mx-auto max-w-conteudo px-page-margin py-section-sm lg:px-page-margin-lg">
        <h1 className="sr-only">{chips.length > 0 ? "Resultados de busca" : "Vitrine"}</h1>
        <div className="gap-grid-gutter lg:grid lg:grid-cols-[15rem_minmax(0,1fr)]">
          {/* A chave remonta o painel a cada URL: voltar no navegador não deixa valor velho nos campos. */}
          <PainelDeFiltros
            key={consulta(estado)}
            estado={estado}
            categorias={categorias}
            faixaInvertida={faixaInvertida}
          />
          <div className="min-w-0 space-y-section-sm">
            <div className="flex flex-wrap items-center justify-between gap-4 max-lg:mt-4">
              <p aria-live="polite" className="font-medium">
                {contagem}
                {estado.termo && <> para &quot;{estado.termo}&quot;</>}
              </p>
              <Ordenar estado={estado} />
            </div>
            {chips.length > 0 && (
              <div className="flex flex-wrap items-center gap-2">
                {chips.map((c) => (
                  <a
                    key={c.rotulo}
                    href={link(estado, c.sem)}
                    aria-label={`Remover filtro ${c.rotulo}`}
                    className="inline-flex max-w-full items-center gap-2 rounded-full border border-border px-3 py-1 text-sm hover:bg-muted"
                  >
                    <span className="truncate">{c.rotulo}</span>
                    <span aria-hidden="true">✕</span>
                  </a>
                ))}
                <a className="text-link text-sm underline" href={limpar}>
                  Limpar filtros
                </a>
              </div>
            )}
            {listagem.total === 0 && chips.length === 0 ? (
              <CatalogoVazio />
            ) : listagem.total === 0 ? (
              <p className="py-section-sm text-center">
                {estado.termo
                  ? `Nenhum Produto encontrado para '${estado.termo}'.`
                  : "Nenhum Produto encontrado com estes filtros."}
              </p>
            ) : listagem.itens.length === 0 ? (
              <p className="py-section-sm text-center">
                Esta página não tem Produtos.{" "}
                <a className="text-link underline" href={link(estado)}>
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
            {paginas > 1 && pagina <= paginas && (
              <Paginacao pagina={pagina} paginas={paginas} href={(p) => link(estado, { pagina: p })} />
            )}
          </div>
        </div>
      </main>
    </Casca>
  );
}

// A primeira, a última e as vizinhas da atual; o resto vira reticências.
// Paginar preserva termo, filtros e ordenação.
function Paginacao({ pagina, paginas, href }: { pagina: number; paginas: number; href: (p: number) => string }) {
  const numeros = [...new Set([1, pagina - 1, pagina, pagina + 1, paginas])]
    .filter((p) => p >= 1 && p <= paginas)
    .sort((a, b) => a - b);
  return (
    <Pagination aria-label="Paginação">
      <PaginationContent>
        {pagina > 1 && (
          <PaginationItem>
            <PaginationPrevious href={href(pagina - 1)} text="Anterior" aria-label="Página anterior" />
          </PaginationItem>
        )}
        {numeros.map((p, i) => [
          i > 0 && p - numeros[i - 1] > 1 && (
            <PaginationItem key={`r${p}`}>
              <PaginationEllipsis />
            </PaginationItem>
          ),
          <PaginationItem key={p}>
            <PaginationLink href={href(p)} isActive={p === pagina} aria-label={`Página ${p}`}>
              {p}
            </PaginationLink>
          </PaginationItem>,
        ])}
        {pagina < paginas && (
          <PaginationItem>
            <PaginationNext href={href(pagina + 1)} text="Próxima" aria-label="Próxima página" />
          </PaginationItem>
        )}
      </PaginationContent>
    </Pagination>
  );
}
