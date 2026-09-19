import { Casca } from "@/components/casca";
import { Skeleton } from "@/components/ui/skeleton";

// O carregamento na forma da página: imagem, título e Caixa — nunca spinner.
export default function Carregando() {
  return (
    <Casca>
      <main className="mx-auto max-w-conteudo space-y-4 px-page-margin py-section-sm lg:px-page-margin-lg" aria-busy="true">
        <Skeleton className="h-4 w-40" />
        <div className="grid gap-grid-gutter lg:grid-cols-[minmax(0,5fr)_minmax(0,4fr)_minmax(0,3fr)] lg:items-start">
          <Skeleton className="aspect-[4/3] w-full rounded-xl" />
          <div className="space-y-4">
            <Skeleton className="h-8 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
            <Skeleton className="h-8 w-1/3" />
            <Skeleton className="h-16 w-full" />
          </div>
          <Skeleton className="h-48 w-full rounded-xl" />
        </div>
      </main>
    </Casca>
  );
}
