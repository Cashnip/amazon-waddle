import Image from "next/image";
import { notFound } from "next/navigation";
import { Card, CardContent } from "@/components/ui/card";
import { Saudacao } from "./saudacao";
import { Comprar } from "./comprar";

// Página de Produto crua da estória 1.5. A composição de marca — Estoque,
// Adicionar ao Carrinho, avaliações — é da Épica 3; aqui o que importa é o
// dado atravessar: Go → Postgres → DTO → tela.
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
};

// O valor viaja em centavos inteiros de ponta a ponta (AD-3); o `R$` é escrito
// aqui, no front, e em nenhum outro lugar. Os dois papéis tipográficos
// monetários trazem o tabular-nums junto.
function Preco({ centavos }: { centavos: number }) {
  const reais = Math.floor(centavos / 100);
  const resto = String(centavos % 100).padStart(2, "0");
  return (
    <p>
      <span className="preco">R$ {reais.toLocaleString("pt-BR")}</span>
      <span className="preco-centavos align-super">{resto}</span>
    </p>
  );
}

export default async function PaginaDeProduto({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  // `no-store`: o preço do Produto não pode vir de cache de construção, e a
  // rota é dinâmica de propósito.
  // encodeURIComponent: o segmento vem da URL e vai para outra URL. Sem
  // escape, um `%2F` normaliza para outra rota do Go — a de mídia devolveria
  // um SVG, e o resposta.json() estouraria em página de erro em vez de 404.
  const resposta = await fetch(`${api}/api/v1/produtos/${encodeURIComponent(id)}`, {
    cache: "no-store",
  });
  if (!resposta.ok) notFound();
  const produto: Produto = await resposta.json();

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

      <main className="mx-auto grid max-w-conteudo gap-grid-gutter px-page-margin py-section-sm md:grid-cols-2 lg:px-page-margin-lg">
        <Card>
          <CardContent>
            <Image
              // unoptimized: o Next 16 recusa otimizar imagem cujo upstream
              // resolva para IP privado, e em compose o Go é exatamente isso.
              unoptimized
              src={produto.imagem_url}
              alt={produto.nome}
              width={800}
              height={800}
              className="h-auto w-full"
            />
          </CardContent>
        </Card>

        <div className="space-y-4">
          <h1 className="text-2xl font-medium">{produto.nome}</h1>
          <p className="text-muted-foreground text-sm">
            Vendido por <span className="text-foreground">{produto.vendedor}</span>
          </p>
          <Preco centavos={produto.preco_centavos} />
          <p className="text-sm">{produto.descricao}</p>
          {/* Da Página de Produto direto ao Pedido, sem Carrinho (Épica 4). */}
          <Comprar produtoId={produto.id} />
        </div>
      </main>
    </div>
  );
}
