"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
} from "@/components/ui/pagination";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Label } from "@/components/ui/label";
import { FALHA_DE_REDE, pedir } from "@/lib/pedir";
import { formatarPreco } from "@/lib/preco";
import {
  ANTIGOS,
  RECENTES,
  SEM_RELATO,
  STATUS,
  anuncioDaTabela,
  enderecoDaTabela,
  filtroDaURL,
  intervaloDaTabela,
  rotaDaTabela,
  rotuloDoStatus,
  statusPorPedido,
  type FiltroDaTabela,
  type PedidoNaTabela,
  type RelatoDaTransicao,
} from "@/lib/pedido";
import { AcoesDoPedido, DesfechoNaTela } from "./acoes-do-pedido";

// A Tabela de Pedidos do Administrador (6.4, FR-32): todos os Pedidos, de
// todos os Compradores, filtráveis por Status e ordenáveis por data. Toda a
// regra é do Go — o filtro, a ordem, o total e `permitidas` vêm prontos —, e o
// envelope é o único de listagem (AD-18).
//
// O estado inteiro vive na URL (`?status=&ordenacao=&pagina=`): recarregar,
// voltar pelo histórico do navegador ou compartilhar o endereço reproduzem a
// mesma lista. Quem lê e monta a URL é `lib/pedido.ts`, testado.

type Listagem = { itens: PedidoNaTabela[]; pagina: number; por_pagina: number; total: number };

// O Select do Radix não aceita valor vazio: "Todos os Status" tem nome próprio.
const TODOS = "todos";

// O instante vem absoluto e em RFC 3339 do servidor; o `dateTime` guarda essa
// forma e o Administrador lê a mesma marca em pt-BR. Instante ilegível não
// vira texto: uma data errada é pior que nenhuma.
function Data({ valor }: { valor?: string }) {
  if (!valor) return null;
  const data = new Date(valor);
  if (Number.isNaN(data.getTime())) return null;
  return (
    <time dateTime={valor} className="whitespace-nowrap">
      {data.toLocaleString("pt-BR", { dateStyle: "short", timeStyle: "short" })}
    </time>
  );
}

export function Pedidos() {
  const router = useRouter();
  const filtro = filtroDaURL(useSearchParams());
  const [lista, setLista] = useState<Listagem | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  // O desfecho da transição vive AQUI, e não na linha: a releitura pode tirar
  // o Pedido da lista (é o que acontece com a Tabela filtrada por Status), e
  // com o Alert dentro da linha ele sumiria junto.
  const [relato, setRelato] = useState<RelatoDaTransicao>(SEM_RELATO);
  // O anúncio da consulta em intervalo: uma região viva para a Tabela inteira,
  // nomeando o Pedido que mudou. Os Status da leitura anterior ficam num ref —
  // são insumo da comparação, e não coisa que a tela desenhe.
  const anteriores = useRef<Record<string, string>>({});
  const [anuncio, setAnuncio] = useState("");

  // Só a consulta mais recente escreve na tela (o contador de
  // `acompanhamento.tsx`): a do intervalo que saiu antes do clique pode voltar
  // depois da releitura forçada, e traria o Status velho de volta por um
  // intervalo inteiro.
  const ultima = useRef(0);

  // A guarda de verdade é o Go: sem Sessão de Administrador o prefixo
  // responde 404, e a tela só a respeita — o mesmo desvio de /admin/produtos.
  const carregar = useCallback(async () => {
    const minha = ++ultima.current;
    try {
      const { resposta, json } = await pedir(rotaDaTabela(filtro), "GET");
      if (minha !== ultima.current) return;
      if (resposta.status === 404) {
        router.replace("/admin/entrar");
        return;
      }
      if (!resposta.ok) {
        setErro(json?.erro?.mensagem ?? "Não foi possível ler os Pedidos.");
        return;
      }
      setErro(null);
      setLista(json);
      setAnuncio(anuncioDaTabela(anteriores.current, json.itens));
      anteriores.current = statusPorPedido(json.itens);
    } catch {
      if (minha === ultima.current) setErro(FALHA_DE_REDE);
    }
    // A consulta depende do filtro inteiro, e não do objeto, que é novo a cada
    // render: sem isto o efeito abaixo recarregaria em laço.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filtro.status, filtro.ordenacao, filtro.pagina, router]);

  useEffect(() => {
    carregar();
  }, [carregar]);

  // A consulta em intervalo (EXPERIENCE): 10 s enquanto a página listar Pedido
  // não terminal. Quem diz que acabou é o `terminal` do Go. O `ref` deixa o
  // intervalo chamar sempre a consulta mais recente sem ser rearmado a cada
  // resposta, o que adiaria a próxima para sempre.
  const consultar = useRef(carregar);
  consultar.current = carregar;
  const ritmo = intervaloDaTabela(lista?.itens ?? []);
  useEffect(() => {
    if (ritmo === null) return;
    const tique = setInterval(() => consultar.current(), ritmo);
    return () => clearInterval(tique);
  }, [ritmo]);

  // Trocar filtro ou ordenação volta à página 1: a página 3 do filtro anterior
  // não é a página 3 deste.
  function irPara(mudanca: Partial<FiltroDaTabela>) {
    router.push(enderecoDaTabela({ ...filtro, pagina: 1, ...mudanca }));
  }

  const paginas = lista ? Math.max(1, Math.ceil(lista.total / lista.por_pagina)) : 1;

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-medium">Pedidos</h1>

      <div className="flex flex-wrap items-end gap-4">
        <div className="space-y-2">
          <Label htmlFor="status">Status</Label>
          <Select
            value={filtro.status || TODOS}
            onValueChange={(v) => irPara({ status: v === TODOS ? "" : v })}
          >
            <SelectTrigger id="status" className="w-56">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={TODOS}>Todos os Status</SelectItem>
              {STATUS.map((s) => (
                <SelectItem key={s} value={s}>
                  {rotuloDoStatus(s)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label htmlFor="ordenacao">Ordenação</Label>
          <Select value={filtro.ordenacao} onValueChange={(v) => irPara({ ordenacao: v })}>
            <SelectTrigger id="ordenacao" className="w-56">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={RECENTES}>Mais recentes primeiro</SelectItem>
              <SelectItem value={ANTIGOS}>Mais antigos primeiro</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {erro && (
        <Alert variant="destructive">
          <AlertDescription>{erro}</AlertDescription>
        </Alert>
      )}

      {/* O desfecho da última transição, fora da Tabela e nas duas regiões
          que existem desde o primeiro render. */}
      <DesfechoNaTela relato={relato} />

      {/* Uma região viva para a Tabela inteira: a consulta de 10 s traz o
          Status novo sem clique nenhum, e o anúncio nomeia o Pedido. Visual
          não — a linha já mostra o Status. */}
      <p className="sr-only" role="status">
        {anuncio}
      </p>

      {/* Antes da primeira resposta há Skeleton, e nunca spinner nem o texto
          de vazio: dizer que não há Pedido enquanto a resposta está a caminho
          seria uma afirmação falsa sobre a loja. */}
      {lista === null && !erro && (
        <div className="space-y-2" aria-busy="true">
          {Array.from({ length: 6 }, (_, i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      )}

      {lista !== null && lista.total === 0 && (
        <p>
          {filtro.status
            ? `Nenhum Pedido em ${rotuloDoStatus(filtro.status)}.`
            : "Nenhum Pedido foi feito ainda."}
        </p>
      )}

      {lista !== null && lista.total > 0 && (
        <>
          {/* A tabela rola dentro do próprio contêiner: a página nunca rola
              na horizontal (UX, responsivo). */}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Número</TableHead>
                <TableHead>Data</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Total</TableHead>
                <TableHead className="text-right">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {lista.itens.map((p) => (
                <TableRow key={p.id}>
                  <TableCell className="font-medium">
                    <a className="text-link underline underline-offset-2" href={`/admin/pedidos/${p.id}`}>
                      {p.numero}
                    </a>
                  </TableCell>
                  <TableCell>
                    <Data valor={p.criado_em} />
                  </TableCell>
                  {/* Sem região viva na célula: uma por linha anunciaria
                      "Separando" sem dizer de qual Pedido. Quem anuncia é a
                      região única da Tabela, acima. */}
                  <TableCell>{rotuloDoStatus(p.status)}</TableCell>
                  <TableCell className="text-right tabular-nums">{formatarPreco(p.total_centavos)}</TableCell>
                  <TableCell>
                    <AcoesDoPedido pedido={p} aoMudar={carregar} aoRelatar={setRelato} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {lista.itens.length === 0 && (
            <p>
              Esta página não tem Pedidos.{" "}
              <a className="text-link underline underline-offset-2" href={enderecoDaTabela({ ...filtro, pagina: 1 })}>
                Voltar ao começo da lista
              </a>
              .
            </p>
          )}
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-muted-foreground text-sm">
              Página {lista.pagina} de {paginas} · {lista.total.toLocaleString("pt-BR")}{" "}
              {lista.total === 1 ? "Pedido" : "Pedidos"}
            </p>
            {paginas > 1 && (
              <Pagination aria-label="Páginas de Pedidos" className="mx-0 w-auto">
                <PaginationContent>
                  {filtro.pagina > 1 && (
                    <PaginationItem>
                      <PaginationLink
                        size="default"
                        href={enderecoDaTabela({ ...filtro, pagina: filtro.pagina - 1 })}
                      >
                        Anterior
                      </PaginationLink>
                    </PaginationItem>
                  )}
                  {filtro.pagina < paginas && (
                    <PaginationItem>
                      <PaginationLink
                        size="default"
                        href={enderecoDaTabela({ ...filtro, pagina: filtro.pagina + 1 })}
                      >
                        Próxima
                      </PaginationLink>
                    </PaginationItem>
                  )}
                </PaginationContent>
              </Pagination>
            )}
          </div>
        </>
      )}
    </div>
  );
}
