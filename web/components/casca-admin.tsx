"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { MenuIcon } from "lucide-react";
import { sair } from "@/lib/sessao";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";

// A casca da área administrativa (UX-DR9): chrome próprio em `chrome-muted`,
// sem busca e sem Carrinho. A guarda de verdade é o Go — o 404 do prefixo
// `/api/v1/admin/` —, e a tela só a respeita: enquanto `GET
// /api/v1/admin/sessao` não responde 200, nada renderiza, e qualquer outra
// resposta manda para o login administrativo.
//
// Os destinos são só os que existem: cada estória acrescenta o seu. Abaixo de
// 640 px eles moram num `Sheet`, aberto por um botão com nome acessível.
const DESTINOS = [
  { href: "/admin/vendedores", rotulo: "Vendedores" },
  { href: "/admin/categorias", rotulo: "Categorias" },
  { href: "/admin/produtos", rotulo: "Produtos" },
];

export function CascaAdmin({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const caminho = usePathname();
  const [nome, setNome] = useState<string | null>(null);

  useEffect(() => {
    let vivo = true;
    fetch("/api/v1/admin/sessao", { credentials: "same-origin" })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null)
      .then((corpo) => {
        if (!vivo) return;
        if (corpo) setNome(corpo.nome);
        else router.replace("/admin/entrar");
      });
    return () => {
      vivo = false;
    };
  }, [router]);

  if (nome === null) return null;

  // O foco visível é escrito à mão pelo mesmo motivo do Menu da conta: o anel
  // padrão do shadcn não bate 3:1 sobre o fundo escuro.
  const foco = "rounded-xs outline-none focus-visible:ring-2 focus-visible:ring-chrome-foreground";
  // A mesma lista na barra (a partir de 640 px) e no Sheet (abaixo disso). O
  // Sheet é claro: lá o anel de foco é o padrão, e não o branco da barra.
  const destinos = (lista: string, anel: string) => (
    <ul className={lista}>
      {DESTINOS.map((d) => (
        <li key={d.href}>
          <a
            className={`${anel} underline-offset-2 hover:underline aria-[current=page]:underline`}
            href={d.href}
            aria-current={caminho === d.href ? "page" : undefined}
          >
            {d.rotulo}
          </a>
        </li>
      ))}
    </ul>
  );
  return (
    <div className="min-h-svh">
      <header className="bg-chrome-muted text-chrome-foreground">
        <div className="mx-auto flex max-w-conteudo flex-wrap items-center justify-between gap-4 px-page-margin py-4 lg:px-page-margin-lg">
          <div className="flex items-center gap-4 sm:gap-6">
            <Sheet>
              <SheetTrigger asChild>
                <button type="button" className={`sm:hidden ${foco}`} aria-label="Abrir a navegação administrativa">
                  <MenuIcon aria-hidden className="size-6" />
                </button>
              </SheetTrigger>
              <SheetContent side="left" showCloseButton={false}>
                <SheetHeader>
                  <SheetTitle>Navegação administrativa</SheetTitle>
                </SheetHeader>
                <nav aria-label="Navegação administrativa" className="px-4">
                  {destinos("space-y-3", "rounded-xs outline-none focus-visible:ring-2 focus-visible:ring-ring")}
                </nav>
                <SheetClose asChild>
                  <button type="button" className="mx-4 self-start text-sm underline underline-offset-2">
                    Fechar
                  </button>
                </SheetClose>
              </SheetContent>
            </Sheet>
            <a className={`wordmark ${foco}`} href="/admin/vendedores">
              azamon
            </a>
            <nav aria-label="Navegação administrativa" className="hidden sm:block">
              {destinos("flex gap-4 text-sm", foco)}
            </nav>
          </div>
          <span className="flex items-center gap-4 text-sm">
            <span>{nome}</span>
            <button type="button" className={`underline underline-offset-2 ${foco}`} onClick={sair}>
              Sair
            </button>
          </span>
        </div>
      </header>

      {children}
    </div>
  );
}
