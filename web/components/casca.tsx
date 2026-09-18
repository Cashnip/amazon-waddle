import { MenuDaConta } from "@/components/menu-da-conta";

// A casca comum das oito telas públicas/Comprador (UX-DR9): o `<header>` que
// hoje estava repetido em seis arquivos vira este componente só — a mudança é
// puramente de composição, o markup e as classes são os mesmos de sempre.
//
// O `<main>` continua em cada página, e não aqui: as larguras divergem
// (`max-w-md`, `max-w-conteudo`, `max-w-2xl`), e cada página passa o seu
// próprio `<main>...</main>` como `children`.
export function Casca({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-svh">
      <header className="bg-chrome text-chrome-foreground">
        <div className="mx-auto flex max-w-conteudo items-center justify-between px-page-margin py-4 lg:px-page-margin-lg">
          <a className="wordmark" href="/">
            azamon
          </a>
          <MenuDaConta />
        </div>
      </header>

      {children}
    </div>
  );
}
