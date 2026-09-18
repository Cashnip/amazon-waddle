"use client";

import { useEffect, useRef, useState } from "react";
import { cn } from "cn";

// A imagem do Produto em 4/3. Sem URL, ou quando o arquivo não carrega, vira
// um bloco neutro com o nome centralizado — nunca o ícone de imagem quebrada.
// `<img>` e não `next/image`: a imagem é SVG embutido no Go, servido na mesma
// origem, e não há o que otimizar (AD-12).
export function ImagemDoProduto({
  src,
  nome,
  className,
  compacta = false,
}: {
  src: string;
  nome: string;
  className?: string;
  // Na miniatura da tabela o nome não cabe, e já está na linha ao lado: o
  // bloco fica mudo.
  compacta?: boolean;
}) {
  // Guarda o `src` que falhou, e não um booleano: trocar a imagem esquece a falha.
  const [falhou, setFalhou] = useState<string | null>(null);
  const img = useRef<HTMLImageElement>(null);

  // Na página que veio do servidor, o erro pode acontecer antes da hidratação,
  // quando o onError ainda não existe. `complete` sem largura é essa falha.
  useEffect(() => {
    const el = img.current;
    if (el && el.complete && el.naturalWidth === 0) setFalhou(src);
  }, [src]);

  const moldura = cn("aspect-[4/3] w-full overflow-hidden rounded-md bg-muted", className);
  if (!src || falhou === src) {
    return (
      <div
        className={cn(moldura, "text-muted-foreground flex items-center justify-center p-2 text-center text-sm")}
        aria-hidden={compacta || undefined}
      >
        {!compacta && nome}
      </div>
    );
  }
  return (
    <img
      ref={img}
      src={src}
      alt={compacta ? "" : nome}
      className={cn(moldura, "object-contain")}
      onError={() => setFalhou(src)}
    />
  );
}
