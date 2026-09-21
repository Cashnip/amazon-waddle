import { notFound } from "next/navigation";
import { Card, CardContent } from "@/components/ui/card";
import { Casca } from "@/components/casca";
import { ImagemDoProduto } from "@/components/imagem-do-produto";
import { Preco } from "@/components/preco";
import { quantidadeDaUrl } from "@/lib/quantidade";
import { CaixaDeCompra } from "./caixa-de-compra";

// A Página de Produto (3.6): breadcrumb da Categoria, informação e a Caixa de
// compra — à direita a partir de 1024 px, empilhada abaixo. A ordem do DOM é a
// da tabulação: título e preço antes da Caixa.
//
// O Server Component não pode usar caminho relativo: `/api/...` só existe no
// navegador, depois do rewrites(). No servidor do Next o destino é o mesmo que
// o next.config.ts usa — o nome de serviço do compose.
const api = (process.env.AZAMON_API_URL || "http://azamon:8080").replace(/\/+$/, "");

type Produto = {
  id: string;
  nome: string;
  descricao: string;
  preco_centavos: number;
  imagem_url: string;
  vendedor: string;
  categoria_id: string;
  categoria: string;
  estoque_disponivel: number;
};

export default async function PaginaDeProduto({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ quantidade?: string | string[]; adicionar?: string | string[] }>;
}) {
  const { id } = await params;
  const { quantidade, adicionar } = await searchParams;
  // `no-store`: preço e disponível não podem vir de cache de construção.
  // encodeURIComponent: o segmento vem da URL e vai para outra URL. Sem
  // escape, um `%2F` normaliza para outra rota do Go — a de mídia devolveria
  // um SVG, e o resposta.json() estouraria em página de erro em vez de 404.
  const resposta = await fetch(`${api}/api/v1/produtos/${encodeURIComponent(id)}`, {
    cache: "no-store",
  });
  // Só o 404 vira "não disponível" (not-found.tsx): inexistente, desativado e
  // de Vendedor desativado são o mesmo caso. Outro erro não é esse, e cai no
  // erro genérico.
  if (resposta.status === 404) notFound();
  if (!resposta.ok) throw new Error(`GET /api/v1/produtos/${id} = ${resposta.status}`);
  const produto: Produto = await resposta.json();

  return (
    <Casca>
      <main className="mx-auto max-w-conteudo space-y-4 px-page-margin py-section-sm lg:px-page-margin-lg">
        <nav aria-label="Trilha de navegação" className="text-sm">
          <ol className="flex flex-wrap items-center gap-2">
            <li>
              <a className="text-link hover:underline" href="/">
                Vitrine
              </a>
            </li>
            <li aria-hidden="true">›</li>
            <li>
              <a className="text-link hover:underline" href={`/?categoria=${produto.categoria_id}`}>
                {produto.categoria}
              </a>
            </li>
          </ol>
        </nav>

        <div className="grid gap-grid-gutter lg:grid-cols-[minmax(0,5fr)_minmax(0,4fr)_minmax(0,3fr)] lg:items-start">
          <Card>
            <CardContent>
              {/* Imagem ausente ou quebrada vira o bloco neutro com o nome. */}
              <ImagemDoProduto src={produto.imagem_url} nome={produto.nome} />
            </CardContent>
          </Card>

          <div className="min-w-0 space-y-4">
            <h1 className="text-2xl font-medium break-words">{produto.nome}</h1>
            <p className="text-muted-foreground text-sm">
              Vendido por <span className="text-foreground">{produto.vendedor}</span>
            </p>
            <Preco centavos={produto.preco_centavos} />
            <p className="text-sm break-words">{produto.descricao}</p>
          </div>

          <CaixaDeCompra
            produtoId={produto.id}
            disponivel={produto.estoque_disponivel}
            quantidadeInicial={quantidadeDaUrl(quantidade, produto.estoque_disponivel)}
            // A volta do Login (4.1): o marcador que a Caixa de compra pôs no `destino`.
            adicionarAoEntrar={adicionar === "1"}
          />
        </div>
      </main>
    </Casca>
  );
}
