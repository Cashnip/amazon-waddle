import { Casca } from "@/components/casca";
import { Skeleton } from "@/components/ui/skeleton";
import { GRADE } from "@/components/cartao-de-produto";

// O carregamento da Vitrine: a grade na forma dos cartões, nunca spinner.
export default function Carregando() {
  return (
    <Casca>
      <main className="mx-auto max-w-conteudo px-page-margin py-section-sm lg:px-page-margin-lg" aria-busy="true">
        <h1 className="sr-only">Vitrine</h1>
        <div className={GRADE}>
          {Array.from({ length: 8 }, (_, i) => (
            <div key={i} className="space-y-3 overflow-hidden rounded-xl ring-1 ring-foreground/10">
              <Skeleton className="aspect-[4/3] w-full rounded-none" />
              <div className="space-y-2 px-4 pb-4">
                <Skeleton className="h-5 w-3/4" />
                <Skeleton className="h-8 w-1/3" />
                <Skeleton className="h-4 w-1/4" />
              </div>
            </div>
          ))}
        </div>
      </main>
    </Casca>
  );
}
