import { Button } from "@/components/ui/button";
import { Casca } from "@/components/casca";

// Inexistente, desativado e de Vendedor desativado são a mesma tela: a
// interface não distingue os três casos. Quem responde 404 é a API; a página
// sai com 200, porque o `loading.tsx` já começou o streaming quando o
// `notFound()` roda, e o Next não troca mais o status.
export default function ProdutoIndisponivel() {
  return (
    <Casca>
      <main className="mx-auto max-w-conteudo space-y-4 px-page-margin py-section-sm lg:px-page-margin-lg">
        <h1 className="text-2xl font-medium">Este Produto não está disponível.</h1>
        <Button asChild size="lg">
          <a href="/">Voltar à Vitrine</a>
        </Button>
      </main>
    </Casca>
  );
}
