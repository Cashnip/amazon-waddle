import { Card, CardContent } from "@/components/ui/card";
import { ImagemDoProduto } from "@/components/imagem-do-produto";
import { Preco } from "@/components/preco";

// A grade da Vitrine: 1 a 4 colunas em 360–639, 640–1023, 1024–1439 e
// ≥1440 px. O `2xl` padrão é 1536, então a quarta coluna vem de um breakpoint
// arbitrário. Mora aqui porque a página e o `loading.tsx` a dividem.
// O quarto degrau é `min-[90rem]` (=1440px), e não `min-[1440px]`: o Tailwind v4
// ordena as variantes de largura por unidade, então a versão em px sai no CSS
// **antes** de `lg:` (64rem) e perdia a cascata para ela — a Vitrine ficava em
// três colunas em qualquer largura. Em rem, o degrau sai depois, e vence.
export const GRADE = "grid grid-cols-1 gap-grid-gutter sm:grid-cols-2 lg:grid-cols-3 min-[90rem]:grid-cols-4";

export type ProdutoDaVitrine = {
  id: string;
  nome: string;
  preco_centavos: number;
  imagem_url: string;
  estoque_disponivel: number;
};

// O Cartão de Produto: o cartão inteiro é o link para a Página de Produto. Sem
// "Adicionar ao Carrinho" até a Épica 4 — com Estoque zero, o "não adicionável"
// é o selo "Indisponível". O verde é só o do selo "Em estoque".
export function CartaoDeProduto({ produto }: { produto: ProdutoDaVitrine }) {
  return (
    <a
      href={`/produtos/${produto.id}`}
      className="group block min-w-0 rounded-xl outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
    >
      <Card className="h-full pt-0">
        <ImagemDoProduto src={produto.imagem_url} nome={produto.nome} className="rounded-none" />
        <CardContent className="space-y-2">
          <h2 className="text-link line-clamp-2 text-base font-medium group-hover:underline">{produto.nome}</h2>
          <Preco centavos={produto.preco_centavos} />
          {produto.estoque_disponivel > 0 ? (
            <p className="text-available text-sm">Em estoque</p>
          ) : (
            <p className="text-muted-foreground text-sm">Indisponível</p>
          )}
        </CardContent>
      </Card>
    </a>
  );
}
