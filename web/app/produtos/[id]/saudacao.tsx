"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";

// É esta ida e volta que fecha a verificação 1 da épica: o cookie emitido pelo
// Go, guardado pelo navegador, volta por caminho relativo através do
// rewrites() e devolve o nome do Comprador. Sem sair da mesma origem, sem CORS.
//
// Sem o menu da conta (2.6), o "Sair" daqui é a única porta para encerrar a
// Sessão. Quem invalida é o DELETE no Go — apagar o cookie no navegador
// deixaria a chave viva no Redis, e o token vazado continuaria valendo.
export function Saudacao() {
  const [nome, setNome] = useState<string | null>(null);

  useEffect(() => {
    let vivo = true;
    fetch("/api/v1/sessao", { credentials: "same-origin" })
      .then((r) => (r.ok ? r.json() : null))
      .then((corpo) => {
        if (vivo) setNome(corpo?.nome ?? null);
      })
      .catch(() => {});
    return () => {
      vivo = false;
    };
  }, []);

  async function sair() {
    // O DELETE é idempotente e responde 204 mesmo sem Sessão: não há erro a
    // exibir, e recarregar é o que faz a casca inteira refletir a saída.
    await fetch("/api/v1/sessao", { method: "DELETE", credentials: "same-origin" }).catch(() => {});
    window.location.reload();
  }

  if (!nome) {
    return (
      <a className="text-link text-sm underline" href="/entrar">
        Entrar
      </a>
    );
  }
  return (
    <span className="flex items-center gap-2 text-sm">
      Olá, {nome}
      <Button variant="link" size="sm" className="px-0" onClick={sair}>
        Sair
      </Button>
    </span>
  );
}
