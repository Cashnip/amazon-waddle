"use client";

import { useEffect, useId, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { paraLogin } from "@/lib/destino";
import { quantidadeMaxima } from "@/lib/quantidade";

// A Caixa de compra (3.6): selo, quantidade e "Adicionar ao Carrinho". O
// número exibido é sempre o Estoque disponível, e a Reserva nunca aparece. A
// Sessão é lida como no Menu da conta, por `GET /api/v1/sessao`.
//
// Sem Sessão, "Adicionar ao Carrinho" leva ao Login com o Produto e a
// quantidade no `destino`; com Sessão, fica desabilitado até o Carrinho da
// Épica 4. O "Confirmar compra" do esqueleto chega como `children`.
export function CaixaDeCompra({
  produtoId,
  disponivel,
  quantidadeInicial,
  children,
}: {
  produtoId: string;
  disponivel: number;
  quantidadeInicial: number;
  children: React.ReactNode;
}) {
  const router = useRouter();
  const id = useId();
  const [quantidade, setQuantidade] = useState(quantidadeInicial);
  // undefined enquanto a Sessão não foi lida: o botão espera, em vez de mandar
  // ao Login quem já entrou.
  const [comSessao, setComSessao] = useState<boolean | undefined>(undefined);

  useEffect(() => {
    let vivo = true;
    fetch("/api/v1/sessao", { credentials: "same-origin" })
      // Só o 401 é "sem Sessão": um 5xx não pode mandar ao Login quem já entrou.
      .then((r) => vivo && setComSessao(r.status === 401 ? false : r.ok ? true : undefined))
      .catch(() => vivo && setComSessao(false));
    return () => {
      vivo = false;
    };
  }, []);

  function adicionar() {
    router.push(paraLogin(`/produtos/${produtoId}?quantidade=${quantidade}`));
  }

  return (
    <Card>
      <CardContent className="space-y-4">
        {disponivel === 0 ? (
          <p className="text-muted-foreground text-sm">Indisponível</p>
        ) : (
          <>
            {/* O verde é só da disponibilidade. Abaixo de 10, o número. */}
            <p className="text-available text-sm">
              {disponivel >= 10 ? "Em estoque" : disponivel === 1 ? "Resta 1 unidade" : `Restam ${disponivel} unidades`}
            </p>
            <div className="flex items-center gap-2">
              <Label htmlFor={id}>Quantidade</Label>
              <Select value={String(quantidade)} onValueChange={(v) => setQuantidade(Number(v))}>
                <SelectTrigger id={id}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Array.from({ length: quantidadeMaxima(disponivel) }, (_, i) => (
                    <SelectItem key={i + 1} value={String(i + 1)}>
                      {i + 1}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Button size="lg" className="w-full" onClick={adicionar} disabled={comSessao !== false}>
                Adicionar ao Carrinho
              </Button>
              {comSessao && <p className="text-muted-foreground text-sm">O Carrinho ainda não está disponível.</p>}
            </div>
            {children}
          </>
        )}
      </CardContent>
    </Card>
  );
}
