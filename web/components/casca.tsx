"use client";

import { Suspense, useEffect, useState } from "react";
import { BuscaGlobal, BuscaGlobalDaUrl, type Categoria } from "@/components/busca-global";
import { CarrinhoNaBarra } from "@/components/carrinho-na-barra";
import { MenuDaConta } from "@/components/menu-da-conta";
import { destinoDaCategoria } from "@/lib/busca";

// A casca comum das telas públicas/Comprador (UX-DR9): a barra superior
// (wordmark · busca global · Carrinho · Menu da conta) e, abaixo, a Faixa de
// Categorias.
//
// As Categorias vêm do navegador, uma vez, no molde do Menu da conta: a Casca
// serve a telas cliente e servidor, e buscar no servidor pediria as
// Categorias por props em toda chamada. Se a carga falha, a busca fica só com
// "Todas" e a Faixa, vazia — sem erro na tela.
//
// O `<main>` continua em cada página, e não aqui: as larguras divergem
// (`max-w-md`, `max-w-conteudo`, `max-w-2xl`), e cada página passa o seu
// próprio `<main>...</main>` como `children`.
export function Casca({ children }: { children: React.ReactNode }) {
  const [categorias, setCategorias] = useState<Categoria[]>([]);

  useEffect(() => {
    let vivo = true;
    fetch("/api/v1/categorias")
      .then((r) => (r.ok ? r.json() : []))
      .then((lista) => {
        if (vivo && Array.isArray(lista)) setCategorias(lista);
      })
      .catch(() => {});
    return () => {
      vivo = false;
    };
  }, []);

  const foco = "rounded-xs outline-none focus-visible:ring-2 focus-visible:ring-chrome-foreground";
  return (
    <div className="min-h-svh">
      <header className="bg-chrome text-chrome-foreground">
        <div className="mx-auto flex max-w-conteudo flex-wrap items-center justify-between gap-x-6 gap-y-3 px-page-margin py-3 lg:px-page-margin-lg">
          <a className={`wordmark ${foco}`} href="/">
            azamon
          </a>
          {/* Abaixo de 640 px, a busca desce para uma linha só dela. */}
          <div className="order-last w-full sm:order-none sm:w-auto sm:flex-1">
            <Suspense fallback={<BuscaGlobal categorias={categorias} />}>
              <BuscaGlobalDaUrl categorias={categorias} />
            </Suspense>
          </div>
          <div className="flex items-center gap-4">
            <CarrinhoNaBarra />
            <MenuDaConta />
          </div>
        </div>
      </header>
      <nav aria-label="Categorias" className="bg-chrome-muted text-chrome-foreground">
        <ul className="mx-auto flex h-10 max-w-conteudo items-center gap-[22px] overflow-x-auto px-page-margin text-[13px] whitespace-nowrap lg:px-page-margin-lg">
          {categorias.map((c) => (
            <li key={c.id}>
              <a className={`${foco} hover:underline`} href={destinoDaCategoria(c.id)}>
                {c.nome}
              </a>
            </li>
          ))}
        </ul>
      </nav>

      {children}
    </div>
  );
}
