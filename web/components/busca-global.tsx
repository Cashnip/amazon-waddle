"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { SearchIcon } from "lucide-react";
import { destinoDaBusca } from "@/lib/busca";

export type Categoria = { id: string; nome: string };

// O foco dentro do pill branco: anel escuro no próprio controle. O anel
// branco do molde do Menu da conta fica no pill inteiro, sobre o chrome.
const FOCO = "outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-chrome";

// A busca global (3.10), com o termo e a Categoria da URL atual. `useSearchParams` pede o
// `<Suspense>` na Casca; o fallback é o formulário vazio.
export function BuscaGlobalDaUrl({ categorias }: { categorias: Categoria[] }) {
  const params = useSearchParams();
  // O mesmo corte do `estadoDe` da Vitrine: o campo nunca mostra além do teto.
  const termo = Array.from((params.get("termo") ?? "").trim()).slice(0, 100).join("");
  const categoria = params.get("categoria")?.toLowerCase() ?? "";
  // O `key` com a querystring recria os campos a cada navegação.
  return <BuscaGlobal key={params.toString()} categorias={categorias} termo={termo} categoria={categoria} />;
}

export function BuscaGlobal(props: { categorias: Categoria[]; termo?: string; categoria?: string }) {
  const router = useRouter();
  const [termo, setTermo] = useState(props.termo ?? "");
  const [categoria, setCategoria] = useState(props.categoria ?? "");
  // Sem a lista (ainda carregando, ou a carga falhou), só "Todas" existe.
  const valida = props.categorias.some((c) => c.id === categoria) ? categoria : "";

  function buscar(e: React.FormEvent) {
    e.preventDefault();
    router.push(destinoDaBusca(termo, valida));
  }

  return (
    <form
      role="search"
      onSubmit={buscar}
      className="flex h-11 w-full min-w-0 overflow-hidden rounded-full bg-background text-foreground has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-chrome-foreground"
    >
      <select
        aria-label="Categoria"
        value={valida}
        onChange={(e) => setCategoria(e.target.value)}
        className={`max-w-28 shrink-0 bg-muted px-3 text-sm sm:max-w-48 ${FOCO}`}
      >
        <option value="">Todas</option>
        {props.categorias.map((c) => (
          <option key={c.id} value={c.id}>
            {c.nome}
          </option>
        ))}
      </select>
      <input
        type="search"
        aria-label="Buscar Produtos"
        placeholder="Buscar Produtos"
        maxLength={100}
        value={termo}
        onChange={(e) => setTermo(e.target.value)}
        className={`min-w-0 flex-1 bg-transparent px-3 placeholder:text-muted-foreground ${FOCO}`}
      />
      <button type="submit" aria-label="Buscar" className={`shrink-0 bg-primary px-5 text-primary-foreground ${FOCO}`}>
        <SearchIcon aria-hidden className="size-5" />
      </button>
    </form>
  );
}
