"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { EnderecoPorExtenso } from "@/components/formulario-de-endereco";
import { enderecoEscolhido, lerEscolha } from "@/lib/checkout";
import { paraLogin } from "@/lib/destino";
import type { Endereco } from "@/lib/endereco";
import { FALHA_DE_REDE, pedir } from "@/lib/pedir";
import { EtapasDoCheckout } from "../etapas";

// ESBOÇO — a Estória 5.4 substitui este arquivo pela Revisão de verdade (Frete,
// total, `Idempotency-Key` e Confirmar Pedido). Aqui só se prova a escolha da
// 5.2: o `id` guardado é conferido contra a lista do Go, e o Endereço mostrado é
// o que o Go devolve, nunca uma cópia guardada no navegador.
//
// Sem escolha guardada, ou com um `id` que não está mais na lista (removido em
// "Meus endereços"), o passo volta ao Endereço.

export function EsbocoDaRevisao() {
  const router = useRouter();
  const [endereco, setEndereco] = useState<Endereco | null>(null);
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

      {!endereco && !erro && (
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
