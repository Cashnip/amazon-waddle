"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { EnderecoPorExtenso } from "@/components/formulario-de-endereco";
import { SeloDoStatus } from "@/components/selo-do-status";
import { FALHA_DE_REDE, pedir } from "@/lib/pedir";
import { formatarPreco } from "@/lib/preco";
import {
  SEM_RELATO,
  TEXTO_APROVADO_SOBRE_CANCELADO,
  TITULO_APROVADO_SOBRE_CANCELADO,
  intervaloDaTabela,
  linhaDoTempo,
  type DetalheAdmin,
  type RelatoDaTransicao,
} from "@/lib/pedido";
import { AcoesDoPedido, DesfechoNaTela } from "../acoes-do-pedido";

// O Detalhe administrativo de um Pedido (6.4, FR-32): quem comprou, o que foi
// comprado com o Vendedor congelado de cada Item, os valores e o Endereço
// congelados, a linha do tempo e o MESMO bloco de ação da Tabela.
//
// Tudo sai de uma chamada só (AD-18). O que esta tela NÃO mostra é de
// propósito: `expira_em`, as Tentativas restantes e o "pode cancelar" são da
// decisão do Comprador, e o Administrador não cancela Pedido (FR-32).

function Instante({ valor }: { valor: string }) {
  const data = new Date(valor);
  const texto = Number.isNaN(data.getTime())
    ? valor
    : data.toLocaleString("pt-BR", { dateStyle: "short", timeStyle: "medium" });
  return <time dateTime={valor}>{texto}</time>;
}

export function DetalheDoPedidoAdmin({ pedidoId }: { pedidoId: string }) {
  const router = useRouter();
  const [pedido, setPedido] = useState<DetalheAdmin | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  // O desfecho da transição vive na superfície, e não no bloco de ação: é a
  // mesma razão da Tabela — a releitura pode remontar o que está abaixo, e o
  // Alert de dentro sumiria com ele.
  const [relato, setRelato] = useState<RelatoDaTransicao>(SEM_RELATO);

  // Só a consulta mais recente escreve na tela (o contador de
  // `acompanhamento.tsx`): a do intervalo que saiu antes do clique pode voltar
  // depois da releitura forçada, e traria o Status velho de volta.
  const ultima = useRef(0);

  const carregar = useCallback(async () => {
    const minha = ++ultima.current;
    try {
      const { resposta, json } = await pedir(`/api/v1/admin/pedidos/${encodeURIComponent(pedidoId)}`, "GET");
      if (minha !== ultima.current) return;
      // 404 é o do prefixo administrativo (Sessão acabada) e o do Pedido que
      // não existe, indistinguíveis de propósito (UX-DR9): a tela vai ao
      // login, que é a saída das duas.
      if (resposta.status === 404) {
        router.replace("/admin/entrar");
        return;
      }
      if (!resposta.ok) {
        setErro(json?.erro?.mensagem ?? "Não foi possível ler o Pedido.");
        return;
      }
      setErro(null);
      setPedido(json);
    } catch {
      if (minha === ultima.current) setErro(FALHA_DE_REDE);
    }
  }, [pedidoId, router]);

  useEffect(() => {
    carregar();
  }, [carregar]);

  // A cadência sai do MESMO helper da Tabela, com uma lista de um: o ritmo e a
  // condição de parar moram num lugar só, e não em dois que divergiriam.
  const consultar = useRef(carregar);
  consultar.current = carregar;
  const ritmo = intervaloDaTabela(pedido ? [pedido] : []);
  useEffect(() => {
    if (ritmo === null) return;
    const tique = setInterval(() => consultar.current(), ritmo);
    return () => clearInterval(tique);
  }, [ritmo]);

  const linhas = pedido ? linhaDoTempo(pedido.historico) : [];

  return (
    <div className="space-y-4">
      <p className="text-sm">
        <a className="text-link underline underline-offset-2" href="/admin/pedidos">
          Voltar à Tabela de Pedidos
        </a>
      </p>

      {erro && (
        <Alert variant="destructive">
          <AlertDescription>{erro}</AlertDescription>
        </Alert>
      )}

      {/* Fora do `pedido &&`: as duas regiões vivas têm de existir desde o
          primeiro render, e a releitura não pode desmontá-las. */}
      <DesfechoNaTela relato={relato} />

      {pedido === null && !erro && (
        <div className="space-y-4" aria-busy="true">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-40 w-full" />
          <Skeleton className="h-40 w-full" />
        </div>
      )}

      {pedido && (
        <>
          <Card>
            <CardContent className="space-y-4">
              <h1 className="text-2xl font-medium">Pedido {pedido.numero}</h1>
              {/* A região vive desde a montagem do Card e só o conteúdo muda:
                  a consulta de 10 s traz o Status novo sem clique nenhum. */}
              <p className="text-sm" role="status">
                Status: <SeloDoStatus status={pedido.status} />
              </p>
              {/* O sinal da FR-26 (6.5), abaixo do Status que ele não muda.
                  Persistente: sem botão de fechar, porque é o registro de um
                  dinheiro aprovado, e não um aviso que se dispensa. Neutro, e
                  não `destructive` nem verde ou laranja: é informação. */}
              {pedido.pagamento_aprovado_sobre_cancelado && (
                <Alert>
                  <AlertTitle>{TITULO_APROVADO_SOBRE_CANCELADO}</AlertTitle>
                  <AlertDescription>{TEXTO_APROVADO_SOBRE_CANCELADO}</AlertDescription>
                </Alert>
              )}
              {/* O mesmo bloco de ação da linha da Tabela: um componente só,
                  para os dois desfechos não divergirem. */}
              <AcoesDoPedido pedido={pedido} aoMudar={carregar} aoRelatar={setRelato} />
              <p className="text-muted-foreground text-sm">
                Última atualização: <Instante valor={pedido.atualizado_em} />
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>
                <h2>Comprador</h2>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-1">
              <p className="font-medium">{pedido.comprador.nome || "—"}</p>
              <p className="text-muted-foreground text-sm">{pedido.comprador.email || "—"}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>
                <h2>Itens do Pedido</h2>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <ul>
                {pedido.itens.map((item, i) => (
                  <li key={item.produto_id}>
                    {i > 0 && <Separator className="my-4" />}
                    <p className="font-medium">{item.nome}</p>
                    {/* O Vendedor é o congelado na compra: quem vendeu, e não
                        quem hoje estaria ligado ao Produto. */}
                    <p className="text-muted-foreground text-sm">Vendedor: {item.vendedor_nome}</p>
                    <p className="text-muted-foreground text-sm tabular-nums">
                      {item.quantidade} × {formatarPreco(item.preco_praticado_centavos)}
                    </p>
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>
                <h2>Histórico</h2>
              </CardTitle>
            </CardHeader>
            <CardContent>
              {/* Uma linha por transição registrada, na ordem do Go, e nenhuma
                  etapa futura — o mesmo desenho do Detalhe do Comprador. */}
              <ol className="rounded-md">
                {linhas.map((linha, i) => (
                  // O histórico só cresce no fim: a posição é chave estável.
                  <li key={i} className="relative pb-4 pl-5 last:pb-0">
                    <span aria-hidden="true" className="bg-background absolute top-1.5 left-0 size-2 rounded-full border" />
                    {i < linhas.length - 1 && (
                      <span aria-hidden="true" className="bg-border absolute top-3.5 -bottom-1.5 left-[3.5px] w-px" />
                    )}
                    <p className="font-medium">{linha.status}</p>
                    {linha.antes && <p className="text-muted-foreground text-sm">antes: {linha.antes}</p>}
                    {linha.motivo && <p className="text-sm">{linha.motivo}</p>}
                    <p className="text-muted-foreground text-sm">
                      <Instante valor={linha.em} />
                    </p>
                  </li>
                ))}
              </ol>
            </CardContent>
          </Card>

          {pedido.endereco && (
            <Card>
              <CardHeader>
                <CardTitle>
                  <h2>Endereço de entrega</h2>
                </CardTitle>
              </CardHeader>
              <CardContent>
                <EnderecoPorExtenso endereco={pedido.endereco} />
              </CardContent>
            </Card>
          )}

          {/* Os três valores são os congelados no Pedido, como o Go os
              devolve: a tela não soma nada (NFR-13). */}
          <Card>
            <CardHeader>
              <CardTitle>
                <h2>Valores</h2>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <dl className="space-y-2 text-sm">
                <div className="flex justify-between gap-4">
                  <dt>Subtotal</dt>
                  <dd className="tabular-nums">{formatarPreco(pedido.subtotal_centavos)}</dd>
                </div>
                <div className="flex justify-between gap-4">
                  <dt>Frete</dt>
                  <dd className="tabular-nums">{formatarPreco(pedido.frete_centavos)}</dd>
                </div>
                <div className="flex justify-between gap-4 border-t pt-2 font-medium">
                  <dt>Total</dt>
                  <dd className="tabular-nums">{formatarPreco(pedido.total_centavos)}</dd>
                </div>
              </dl>
            </CardContent>
          </Card>

          <Button asChild variant="outline">
            <a href="/admin/pedidos">Voltar à Tabela de Pedidos</a>
          </Button>
        </>
      )}
    </div>
  );
}
