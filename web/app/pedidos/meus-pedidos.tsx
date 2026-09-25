"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Pagination, PaginationContent, PaginationItem, PaginationLink } from "@/components/ui/pagination";
import { Skeleton } from "@/components/ui/skeleton";
import { Preco } from "@/components/preco";
import { SeloDoStatus } from "@/components/selo-do-status";
import { paraLogin } from "@/lib/destino";
import { FALHA_DE_REDE, pedir } from "@/lib/pedir";

// "Meus pedidos" (6.1): número, data, total e Status, do mais recente para o
// mais antigo e só os do Comprador autenticado — o Go filtra pelo dono na
// própria consulta (AD-11) e ordena lá, então a tela nunca reordena nem
// filtra. O envelope é o único de listagem (AD-18).
//
// O Status sai como texto, pelo ponto único de `lib/pedido.ts`; o selo (cor e
// forma) é da 6.7, que o fecha nas três superfícies de uma vez.
// `criado_em` é opcional porque o Go o emite com `omitempty`: `saidaPedido` é
// compartilhada com a criação e o Detalhe, que não leem o instante. Na
// listagem ele vem sempre; declará-lo obrigatório faria a coluna sumir sem a
// tela nem saber.
type ItemDaLista = {
  id: string;
  numero: string;
  status: string;
  total_centavos: number;
  criado_em?: string;
};

type Listagem = { itens: ItemDaLista[]; pagina: number; por_pagina: number; total: number };

// O instante vem absoluto e em RFC 3339 do servidor; o `dateTime` guarda essa
// forma e o Comprador lê a mesma marca em pt-BR. Instante que não se lê não
// vira texto: uma data errada é pior que nenhuma.
function Data({ valor }: { valor?: string }) {
  if (!valor) return null;
  const data = new Date(valor);
  if (Number.isNaN(data.getTime())) return null;
  return (
    <time dateTime={valor} className="text-muted-foreground text-sm">
      {data.toLocaleDateString("pt-BR")}
    </time>
  );
}

export function MeusPedidos() {
  const router = useRouter();
  // A página mora na URL: recarregar a tela — ou voltar a ela pelo histórico
  // do navegador — continua na mesma página (o padrão de admin/produtos).
  const pagina = Math.max(1, Number.parseInt(useSearchParams().get("pagina") ?? "1", 10) || 1);
  const [lista, setLista] = useState<Listagem | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  // Sessão expirada leva ao Login com o caminho atual no `destino`: a tela
  // não tem o que mostrar sem Sessão, e o Comprador volta a esta mesma lista
  // depois de entrar.
  const carregar = useCallback(async () => {
    try {
      const { resposta, json } = await pedir(`/api/v1/pedidos?pagina=${pagina}`, "GET");
      if (resposta.status === 401) {
        router.push(paraLogin());
        return;
      }
      if (!resposta.ok) {
        setErro(json?.erro?.mensagem ?? "Não foi possível ler os Pedidos.");
        return;
      }
      setErro(null);
      setLista(json);
    } catch {
      setErro(FALHA_DE_REDE);
    }
  }, [router, pagina]);

  useEffect(() => {
    carregar();
  }, [carregar]);

  const paginas = lista ? Math.max(1, Math.ceil(lista.total / lista.por_pagina)) : 1;

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-medium">Meus pedidos</h1>

      {erro && (
        <Alert variant="destructive">
          <AlertDescription>{erro}</AlertDescription>
        </Alert>
      )}

      {/* Antes da primeira resposta há Skeleton, e nunca spinner nem o texto
          de vazio: dizer que a conta não tem Pedido enquanto a resposta está a
          caminho seria uma afirmação falsa sobre a conta. */}
      {lista === null && !erro && (
        <div className="space-y-4" aria-busy="true">
          {Array.from({ length: 4 }, (_, i) => (
            <Card key={i}>
              <CardContent className="flex flex-wrap items-center justify-between gap-4">
                <Skeleton className="h-5 w-32" />
                <Skeleton className="h-5 w-20" />
                <Skeleton className="h-5 w-28" />
                <Skeleton className="h-7 w-24" />
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* O estado vazio de verdade: a conta não tem nenhum Pedido, e a saída é
          uma só — a Vitrine. Só na primeira página: página além do total é
          lista vazia de uma conta que tem Pedidos, e ali a saída é voltar. */}
      {lista !== null && lista.total === 0 && (
        <Card>
          <CardContent className="space-y-4 text-center">
            <p>Você ainda não fez nenhum Pedido.</p>
            <Button asChild>
              <a href="/">Voltar à Vitrine</a>
            </Button>
          </CardContent>
        </Card>
      )}

      {lista !== null && lista.total > 0 && lista.itens.length === 0 && (
        <Card>
          <CardContent className="space-y-4 text-center">
            <p>Esta página não tem Pedidos.</p>
            <Button variant="outline" asChild>
              <a href="?pagina=1">Voltar ao começo da lista</a>
            </Button>
          </CardContent>
        </Card>
      )}

      {lista?.itens.map((pedido) => (
        <Card key={pedido.id}>
          <CardContent>
            {/* A linha inteira é um `<a>` só: o alvo de toque é o cartão, e
                não o número sozinho. Grade de colunas iguais, para número,
                data, selo e total caírem na mesma coluna em toda linha; só o
                número é sublinhado. */}
            <a
              className="grid grid-cols-2 items-center justify-items-start gap-x-4 gap-y-1 tabular-nums sm:grid-cols-4"
              href={`/pedidos/${pedido.id}`}
            >
              <span className="text-link font-medium underline underline-offset-2">{pedido.numero}</span>
              <Data valor={pedido.criado_em} />
              <SeloDoStatus status={pedido.status} />
              <span className="justify-self-end">
                <Preco centavos={pedido.total_centavos} />
              </span>
            </a>
          </CardContent>
        </Card>
      ))}

      {/* Rodapé e navegação só com a página cheia: além da última, "Página 9
          de 1" seria falso e o "Anterior" levaria a outra página vazia — a
          volta é do cartão acima, e é uma só. */}
      {lista !== null && lista.itens.length > 0 && (
        <div className="flex flex-wrap items-center justify-between gap-2">
          <p className="text-muted-foreground text-sm">
            Página {lista.pagina} de {paginas} · {lista.total.toLocaleString("pt-BR")}{" "}
            {lista.total === 1 ? "Pedido" : "Pedidos"}
          </p>
          {paginas > 1 && (
            <Pagination aria-label="Páginas de Pedidos" className="mx-0 w-auto">
              <PaginationContent>
                {pagina > 1 && (
                  <PaginationItem>
                    <PaginationLink size="default" href={`?pagina=${pagina - 1}`}>
                      Anterior
                    </PaginationLink>
                  </PaginationItem>
                )}
                {pagina < paginas && (
                  <PaginationItem>
                    <PaginationLink size="default" href={`?pagina=${pagina + 1}`}>
                      Próxima
                    </PaginationLink>
                  </PaginationItem>
                )}
              </PaginationContent>
            </Pagination>
          )}
        </div>
      )}
    </div>
  );
}
