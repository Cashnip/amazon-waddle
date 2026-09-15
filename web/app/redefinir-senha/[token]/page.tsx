import { Redefinir } from "./redefinir";

// Tela de redefinição da estória 2.3 (UX-DR20a). O token vem do caminho, que é
// o próprio link que chegou ao Comprador.
//
// O Server Component não valida nada: quem decide se o token vale é o Go, e no
// mesmo PUT que troca a senha — perguntar antes seria uma ida a mais e uma
// janela entre a pergunta e a resposta. É o filho `"use client"` que fala com
// o Go por caminho relativo, como na tela do Pedido.
export default async function PaginaDeRedefinicao({
  params,
}: {
  params: Promise<{ token: string }>;
}) {
  const { token } = await params;

  return (
    <div className="min-h-svh">
      <header className="bg-chrome text-chrome-foreground">
        <div className="mx-auto flex max-w-conteudo items-center px-page-margin py-4 lg:px-page-margin-lg">
          <a className="wordmark" href="/">
            azamon
          </a>
        </div>
      </header>

      <main className="mx-auto max-w-md px-page-margin py-section-sm">
        <Redefinir token={token} />
      </main>
    </div>
  );
}
