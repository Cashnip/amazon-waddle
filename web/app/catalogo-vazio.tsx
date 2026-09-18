"use client";

import { useEffect, useState } from "react";

// O Catálogo vazio: o caminho para cadastrar aparece só para a Sessão de
// Administrador, que é o que `GET /api/v1/admin/sessao` responde com 200.
export function CatalogoVazio() {
  const [administrador, setAdministrador] = useState(false);

  useEffect(() => {
    let vivo = true;
    fetch("/api/v1/admin/sessao", { credentials: "same-origin" })
      .then((r) => {
        if (vivo) setAdministrador(r.ok);
      })
      .catch(() => {});
    return () => {
      vivo = false;
    };
  }, []);

  return (
    <div className="space-y-2 py-section-sm text-center">
      <p>O Catálogo ainda não tem Produtos.</p>
      {administrador && (
        <a className="text-link text-sm underline" href="/admin/produtos">
          Cadastrar Produtos
        </a>
      )}
    </div>
  );
}
