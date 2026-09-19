"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { EnderecoPorExtenso } from "@/components/formulario-de-endereco";
import { type Cotacao, enderecoEscolhido, freteGratis, lerEscolha, rotaDoFrete } from "@/lib/checkout";
import { paraLogin } from "@/lib/destino";
import type { Endereco } from "@/lib/endereco";
import { FALHA_DE_REDE, pedir } from "@/lib/pedir";
import { formatarPreco } from "@/lib/preco";
import { EtapasDoCheckout } from "../etapas";

// ESBOÇO — a Estória 5.4 substitui este arquivo pela Revisão de verdade (os
// Itens, `Idempotency-Key` e Confirmar Pedido). Aqui só se prova a escolha da
// 5.2: o `id` guardado é conferido contra a lista do Go, e o Endereço mostrado é
// o que o Go devolve, nunca uma cópia guardada no navegador.
//
// A 5.3 acrescentou o Frete: subtotal, Frete e total vêm prontos do Go a cada
// abertura, para o Endereço escolhido. A tela não soma e não conhece região.
//
// Sem escolha guardada, ou com um `id` que não está mais na lista (removido em
// "Meus endereços"), o passo volta ao Endereço.

export function EsbocoDaRevisao() {
  const router = useRouter();
  const [endereco, setEndereco] = useState<Endereco | null>(null);
  const [cotacao, setCotacao] = useState<Cotacao | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  useEffect(() => {
    let vivo = true;
    const guardado = lerEscolha();
    if (guardado === null) {
      router.replace("/checkout/endereco");
      return;
    }
    (async () => {
      try {
        const { resposta, json } = await pedir("/api/v1/enderecos", "GET");
        if (!vivo) return;
        if (resposta.status === 401) {
          router.push(paraLogin());
          return;
        }
        if (!resposta.ok) {
          setErro(json?.erro?.mensagem ?? "Não foi possível ler os Endereços.");
          return;
        }
        const escolhido = enderecoEscolhido<Endereco>(json ?? [], guardado);
        if (!escolhido) {
          router.replace("/checkout/endereco");
          return;
        }
        setEndereco(escolhido);

        const frete = await pedir(rotaDoFrete(escolhido.id), "GET");
        if (!vivo) return;
        if (frete.resposta.status === 401) {
          router.push(paraLogin());
          return;
        }
        // O Endereço removido entre a lista e a cotação é o mesmo caso do `id`
        // fora da lista: volta ao passo Endereço, e não fica num beco sem saída.
        if (frete.resposta.status === 404) {
          router.replace("/checkout/endereco");
          return;
        }
        if (!frete.resposta.ok || !frete.json) {
          setErro(frete.json?.erro?.mensagem ?? "Não foi possível calcular o Frete.");
          return;
        }
        setCotacao(frete.json);
      } catch {
        if (vivo) setErro(FALHA_DE_REDE);
      }
    })();
    return () => {
      vivo = false;
    };
  }, [router]);

  return (
    <div className="space-y-4">
      <EtapasDoCheckout atual="Revisão" />
      <h1 className="text-2xl font-medium">Revisão do Pedido</h1>

      {erro && (
        <Alert variant="destructive">
          <AlertDescription>{erro}</AlertDescription>
        </Alert>
      )}

      {(!endereco || !cotacao) && !erro && (
        <p className="text-muted-foreground text-sm" role="status">
          Carregando a Revisão…
        </p>
      )}

      {endereco && (
        <Card>
          <CardHeader>
            <CardTitle>
              <h2>Endereço de entrega</h2>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <EnderecoPorExtenso endereco={endereco} />
          </CardContent>
        </Card>
      )}

      {cotacao && (
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
                <dd className="tabular-nums">{formatarPreco(cotacao.subtotal_centavos)}</dd>
              </div>
              <div className="flex justify-between gap-4">
                <dt>
                  Frete <span className="text-muted-foreground">({cotacao.regiao})</span>
                </dt>
                <dd className="tabular-nums">{freteGratis(cotacao) ? "Grátis" : formatarPreco(cotacao.frete_centavos)}</dd>
              </div>
              <div className="flex justify-between gap-4 border-t pt-2 font-medium">
                <dt>Total</dt>
                <dd className="tabular-nums">{formatarPreco(cotacao.total_centavos)}</dd>
              </div>
            </dl>
          </CardContent>
        </Card>
      )}

      <div className="flex flex-wrap gap-2">
        <Button variant="outline" asChild>
          <a href="/checkout/endereco">Trocar o Endereço</a>
        </Button>
        <Button variant="outline" asChild>
          <a href="/carrinho">Voltar ao Carrinho</a>
        </Button>
      </div>
    </div>
  );
}
