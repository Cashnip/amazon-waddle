"use client";

import { useId, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { centavosParaReais, reaisParaCentavos } from "@/lib/preco";

export type Categoria = { id: string; nome: string };

// O estado da listagem como a URL o traz (centavos, texto cru).
export type Estado = {
  termo: string;
  categoria: string;
  preco_min: string;
  preco_max: string;
  ordenacao: string;
};

const INVERTIDA = "O preço mínimo não pode ser maior que o máximo.";

// A URL de destino: vazios saem, e a página nunca vai — mudar filtro, termo
// ou ordenação volta para a página 1.
function destino(estado: Estado) {
  const q = new URLSearchParams();
  for (const [chave, valor] of Object.entries(estado)) if (valor) q.set(chave, valor);
  const texto = q.toString();
  return texto ? `/?${texto}` : "/";
}

// O painel de filtros: fixo à esquerda a partir de 1024 px e `Sheet` abaixo.
export function PainelDeFiltros(props: { estado: Estado; categorias: Categoria[]; faixaInvertida: boolean }) {
  const [aberto, setAberto] = useState(false);
  return (
    <>
      <aside className="hidden lg:block" aria-label="Filtros">
        <Formulario {...props} />
      </aside>
      <Sheet open={aberto} onOpenChange={setAberto}>
        <SheetTrigger asChild>
          <Button variant="outline" className="lg:hidden">
            Filtros
          </Button>
        </SheetTrigger>
        <SheetContent side="left" className="overflow-y-auto">
          <SheetHeader>
            <SheetTitle>Filtros</SheetTitle>
          </SheetHeader>
          <div className="px-4 pb-4">
            <Formulario {...props} aoNavegar={() => setAberto(false)} />
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
}

function Formulario({
  estado,
  categorias,
  faixaInvertida,
  aoNavegar,
}: {
  estado: Estado;
  categorias: Categoria[];
  faixaInvertida: boolean;
  aoNavegar?: () => void;
}) {
  const router = useRouter();
  const id = useId();
  const reais = (c: string) => (/^\d+$/.test(c) ? centavosParaReais(Number(c)) : "");
  const [termo, setTermo] = useState(estado.termo);
  const [categoria, setCategoria] = useState(estado.categoria);
  const [minimo, setMinimo] = useState(reais(estado.preco_min));
  const [maximo, setMaximo] = useState(reais(estado.preco_max));
  const [erro, setErro] = useState(faixaInvertida ? INVERTIDA : "");

  function aplicar(e: React.FormEvent) {
    e.preventDefault();
    const min = minimo.trim() ? reaisParaCentavos(minimo) : undefined;
    const max = maximo.trim() ? reaisParaCentavos(maximo) : undefined;
    if (min === null || max === null) return setErro("Informe o preço em reais, como 12,90.");
    // Faixa invertida: a mensagem fica em linha, e nada é navegado.
    if (min !== undefined && max !== undefined && min > max) return setErro(INVERTIDA);
    setErro("");
    aoNavegar?.();
    router.push(
      destino({
        termo: termo.trim(),
        categoria,
        preco_min: min === undefined ? "" : String(min),
        preco_max: max === undefined ? "" : String(max),
        ordenacao: estado.ordenacao,
      }),
    );
  }

  const idErro = `${id}-erro`;
  return (
    <form onSubmit={aplicar} className="space-y-6">
      <div className="space-y-2">
        <Label htmlFor={`${id}-termo`}>Buscar Produtos</Label>
        <Input
          id={`${id}-termo`}
          type="search"
          maxLength={100}
          value={termo}
          onChange={(e) => setTermo(e.target.value)}
        />
      </div>

      <fieldset className="space-y-2">
        <legend className="mb-2 text-sm font-medium">Categoria</legend>
        {[{ id: "", nome: "Todas" }, ...categorias].map((c) => (
          <label key={c.id || "todas"} className="flex items-center gap-2 text-sm">
            <input
              type="radio"
              name={`${id}-categoria`}
              value={c.id}
              checked={categoria === c.id}
              onChange={() => setCategoria(c.id)}
            />
            {c.nome}
          </label>
        ))}
      </fieldset>

      <fieldset className="space-y-2">
        <legend className="mb-2 text-sm font-medium">Faixa de preço (R$)</legend>
        <div className="flex items-center gap-2">
          <Input
            aria-label="Preço mínimo"
            inputMode="decimal"
            placeholder="Mín."
            value={minimo}
            aria-invalid={!!erro || undefined}
            aria-describedby={erro ? idErro : undefined}
            onChange={(e) => setMinimo(e.target.value)}
          />
          <span aria-hidden="true">a</span>
          <Input
            aria-label="Preço máximo"
            inputMode="decimal"
            placeholder="Máx."
            value={maximo}
            aria-invalid={!!erro || undefined}
            aria-describedby={erro ? idErro : undefined}
            onChange={(e) => setMaximo(e.target.value)}
          />
        </div>
        {erro && (
          <p id={idErro} className="text-destructive text-sm" role="alert">
            {erro}
          </p>
        )}
      </fieldset>

      <Button type="submit" className="w-full">
        Aplicar filtros
      </Button>
    </form>
  );
}

const ORDENACOES = [
  { valor: "recentes", rotulo: "Mais recentes" },
  { valor: "preco_asc", rotulo: "Preço: menor primeiro" },
  { valor: "preco_desc", rotulo: "Preço: maior primeiro" },
];

// "Ordenar por": troca a ordenação e volta para a página 1.
export function Ordenar({ estado }: { estado: Estado }) {
  const router = useRouter();
  const id = useId();
  return (
    <div className="flex items-center gap-2">
      <Label htmlFor={id}>Ordenar por</Label>
      <Select
        value={estado.ordenacao || "recentes"}
        onValueChange={(ordenacao) => router.push(destino({ ...estado, ordenacao }))}
      >
        <SelectTrigger id={id}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {ORDENACOES.map((o) => (
            <SelectItem key={o.valor} value={o.valor}>
              {o.rotulo}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
