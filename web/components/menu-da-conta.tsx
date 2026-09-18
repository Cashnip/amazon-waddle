"use client";

import { useEffect, useState } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { paraLogin } from "@/lib/destino";
import { sair } from "@/lib/sessao";

// O Menu da conta (UX-DR8/UX-DR9) substitui a Saudacao crua da 1.5 nas oito
// telas públicas/Comprador. Mesma ida e volta de sempre: `GET /api/v1/sessao`
// pelo caminho relativo — é o cookie atravessando o rewrites() — e o mesmo
// `sair()` de `lib/sessao.ts`, que o Perfil também usa no seu próprio botão.
//
// `DropdownMenu` do shadcn entra sem alteração (UX-DR1): a composição é só
// Root/Trigger/Content/Item/Separator, no molde do arquivo em
// `components/ui/dropdown-menu.tsx`.
type Sessao = { nome: string; email: string };

export function MenuDaConta() {
  const [sessao, setSessao] = useState<Sessao | null>(null);
  // O destino nasce em "/entrar" liso, e só vira paraLogin() depois de montar:
  // paraLogin() lê window.location, que não existe durante a renderização no
  // servidor — chamá-la no corpo do componente quebraria a primeira pintura
  // de toda página que usa a Casca.
  const [destino, setDestino] = useState("/entrar");

  useEffect(() => {
    setDestino(paraLogin());

    let vivo = true;
    fetch("/api/v1/sessao", { credentials: "same-origin" })
      .then((r) => (r.ok ? r.json() : null))
      .then((corpo) => {
        if (vivo) setSessao(corpo ? { nome: corpo.nome, email: corpo.email } : null);
      })
      .catch(() => {});
    return () => {
      vivo = false;
    };
  }, []);

  // Sem Sessão: Entrar (preservando o destino) e Criar conta, sem dropdown.
  // O foco visível aqui é escrito à mão, e não herdado do ring padrão do
  // shadcn (pensado para superfície clara): estes dois links vivem direto
  // sobre `bg-chrome`, escuro, e o anel cinza-claro de sempre não bate os 3:1
  // do NFR-11/UX-DR14 nesse fundo.
  if (!sessao) {
    return (
      <span className="flex items-center gap-4 text-sm">
        <a
          className="rounded-xs underline underline-offset-2 outline-none focus-visible:ring-2 focus-visible:ring-chrome-foreground"
          href={destino}
        >
          Entrar
        </a>
        <a
          className="rounded-xs underline underline-offset-2 outline-none focus-visible:ring-2 focus-visible:ring-chrome-foreground"
          href="/cadastrar"
        >
          Criar conta
        </a>
      </span>
    );
  }

  // Com Sessão: o DropdownMenu com os quatro destinos (UX-DR9).
  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="rounded-xs text-sm outline-none focus-visible:ring-2 focus-visible:ring-chrome-foreground">
        Olá, {sessao.nome}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem asChild>
          <a href="/pedidos">Meus pedidos</a>
        </DropdownMenuItem>
        <DropdownMenuItem asChild>
          <a href="/enderecos">Meus endereços</a>
        </DropdownMenuItem>
        <DropdownMenuItem asChild>
          <a href="/perfil">Perfil</a>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={sair}>Sair</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
