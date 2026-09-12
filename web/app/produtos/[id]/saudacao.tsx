"use client";

import { useEffect, useState } from "react";

// É esta ida e volta que fecha a verificação 1 da épica: o cookie emitido pelo
// Go, guardado pelo navegador, volta por caminho relativo através do
// rewrites() e devolve o nome do Comprador. Sem sair da mesma origem, sem CORS.
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

  if (!nome) {
    return (
      <a className="text-link text-sm underline" href="/entrar">
        Entrar
      </a>
    );
  }
  return <span className="text-sm">Olá, {nome}</span>;
}
