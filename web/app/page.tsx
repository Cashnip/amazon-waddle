import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";

// Página de conferência da casca. Existe para que a base seja inspecionável
// a olho nu — dez tokens, três papéis, um componente do shadcn. As dezesseis
// composições da UX-DR8 nascem nas épicas que as usam, não aqui.

const TOKENS = [
  { nome: "chrome", hex: "#131921", classe: "bg-chrome" },
  { nome: "chrome-foreground", hex: "#FFFFFF", classe: "bg-chrome-foreground" },
  { nome: "chrome-muted", hex: "#232F3E", classe: "bg-chrome-muted" },
  { nome: "primary", hex: "#FFD814", classe: "bg-primary" },
  { nome: "primary-foreground", hex: "#0F1111", classe: "bg-primary-foreground" },
  { nome: "primary-strong", hex: "#FFA41C", classe: "bg-primary-strong" },
  { nome: "primary-strong-foreground", hex: "#0F1111", classe: "bg-primary-strong-foreground" },
  { nome: "link", hex: "#007185", classe: "bg-link" },
  { nome: "available", hex: "#007600", classe: "bg-available" },
  { nome: "available-foreground", hex: "#FFFFFF", classe: "bg-available-foreground" },
];

export default function Conferencia() {
  return (
    <div className="min-h-svh">
      <header className="bg-chrome text-chrome-foreground">
        <div className="mx-auto flex max-w-conteudo items-center px-page-margin py-4 lg:px-page-margin-lg">
          <span className="wordmark">azamon</span>
        </div>
      </header>
      <div className="bg-chrome-muted text-chrome-foreground">
        <h1 className="mx-auto max-w-conteudo px-page-margin py-2 text-sm lg:px-page-margin-lg">
          Conferência da casca — estória 1.2
        </h1>
      </div>

      <main className="mx-auto max-w-conteudo space-y-section-sm px-page-margin py-section-sm lg:space-y-section lg:px-page-margin-lg lg:py-section">
        <Card>
          <CardHeader>
            <CardTitle>
              <h2>Os dez tokens de marca</h2>
            </CardTitle>
          </CardHeader>
          <CardContent className="grid grid-cols-1 gap-grid-gutter sm:grid-cols-2 lg:grid-cols-3">
            {TOKENS.map((t) => (
              <div key={t.nome} className="flex items-center gap-4">
                <div className={`size-12 shrink-0 rounded-md border ${t.classe}`} />
                <div className="text-sm">
                  <div className="font-medium">{t.nome}</div>
                  <div className="text-muted-foreground tabular-nums">{t.hex}</div>
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>
              <h2>Os três papéis tipográficos</h2>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-grid-gutter">
            <div>
              <span className="wordmark">azamon</span>
              <p className="text-muted-foreground text-sm">wordmark — 22px / 700 / -0.02em</p>
            </div>
            <Separator />
            <div>
              <span className="preco">R$ 1.299</span>
              <span className="preco-centavos align-super">90</span>
              <p className="text-muted-foreground text-sm">
                preco 28px + preco-centavos 14px, ambos em tabular-nums
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>
              <h2>Componente do shadcn, sem alteração</h2>
            </CardTitle>
          </CardHeader>
          <CardContent className="flex flex-wrap items-center gap-4">
            <Badge>Badge padrão</Badge>
            <Badge variant="secondary">AGUARDANDO_PAGAMENTO</Badge>
            <Badge className="bg-available text-available-foreground rounded-full">
              ENTREGUE
            </Badge>
            <a className="text-link text-sm underline" href="/api/v1/saude">
              /api/v1/saude pelo rewrite
            </a>
          </CardContent>
        </Card>
      </main>
    </div>
  );
}
