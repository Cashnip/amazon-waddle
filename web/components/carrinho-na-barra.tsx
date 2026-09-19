"use client";

import { useEffect, useState } from "react";
import { ShoppingCartIcon } from "lucide-react";
import { CARRINHO_ALTERADO } from "@/lib/carrinho";
import { paraLogin } from "@/lib/destino";

// O ícone do Carrinho na barra superior (UX-DR9), com o contador de unidades,
// e não de linhas. Na barra do Comprador, e em nenhum lugar da área
// administrativa.
//
// Mesma ida e volta de sempre: `GET /api/v1/carrinho` por caminho relativo, com
// o cookie atravessando o rewrites(). O 401 é "sem Sessão": o ícone fica sem
// número e leva ao Login, que volta ao Carrinho. Qualquer outra falha deixa o
// ícone sem número, e não vira erro na tela — o Carrinho abre e diz o que
// houve. A leitura acontece ao montar e a cada aviso da tela ou da Caixa de
// compra, sem consulta em intervalo.
export function CarrinhoNaBarra() {
  // `undefined` até a primeira resposta, e depois de uma falha que não é 401.
  const [semSessao, setSemSessao] = useState<boolean | undefined>(undefined);
  const [unidades, setUnidades] = useState<number | undefined>(undefined);

  useEffect(() => {
    let vivo = true;
    // Duas leituras seguidas podem voltar fora de ordem: só a mais recente
    // escreve, senão o contador ficaria com o número da anterior.
    let ultima = 0;
    function ler() {
      const minha = ++ultima;
      fetch("/api/v1/carrinho", { credentials: "same-origin" })
        .then(async (r) => {
          if (!vivo || minha !== ultima) return;
          if (r.status === 401) {
            setSemSessao(true);
            setUnidades(undefined);
            return;
          }
          const corpo = r.ok ? await r.json() : null;
          if (!vivo || minha !== ultima) return;
          setSemSessao(false);
          setUnidades(typeof corpo?.unidades === "number" ? corpo.unidades : undefined);
        })
        .catch(() => {});
    }
    ler();
    window.addEventListener(CARRINHO_ALTERADO, ler);
    return () => {
      vivo = false;
      window.removeEventListener(CARRINHO_ALTERADO, ler);
    };
  }, []);

  const rotulo =
    unidades === undefined
      ? "Carrinho"
      : `Carrinho, ${unidades === 1 ? "1 unidade" : `${unidades} unidades`}`;
  return (
    <a
      className="inline-flex items-center gap-1 rounded-xs text-sm outline-none focus-visible:ring-2 focus-visible:ring-chrome-foreground"
      href={semSessao ? paraLogin("/carrinho") : "/carrinho"}
      aria-label={rotulo}
    >
      <ShoppingCartIcon aria-hidden="true" className="size-5" />
      {unidades !== undefined && (
        <span aria-hidden="true" className="min-w-4 text-center font-medium tabular-nums">
          {unidades}
        </span>
      )}
    </a>
  );
}
