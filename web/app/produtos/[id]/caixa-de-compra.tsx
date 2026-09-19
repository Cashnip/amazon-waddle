"use client";

import { useEffect, useId, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { avisarCarrinhoAlterado } from "@/lib/carrinho";
import { paraLogin } from "@/lib/destino";
import { FALHA_DE_REDE, pedir } from "@/lib/pedir";
import { quantidadeMaxima } from "@/lib/quantidade";

// A Caixa de compra (3.6): selo, quantidade e "Adicionar ao Carrinho". O
// número exibido é sempre o Estoque disponível, e a Reserva nunca aparece. A
// Sessão é lida como no Menu da conta, por `GET /api/v1/sessao`.
//
// Com Sessão, "Adicionar ao Carrinho" grava direto (4.1), sem atualização
// otimista. Sem Sessão, leva ao Login com o Produto, a quantidade e o marcador
// `adicionar` no `destino`; a volta (`adicionarAoEntrar`) cria o Item uma vez e
// tira o marcador da URL, para o recarregar não somar de novo. O "Confirmar
// o Pedido" do esqueleto chega como `children`.
export function CaixaDeCompra({
  produtoId,
  disponivel,
  quantidadeInicial,
  adicionarAoEntrar,
  children,
}: {
  produtoId: string;
  disponivel: number;
  quantidadeInicial: number;
  adicionarAoEntrar: boolean;
  children: React.ReactNode;
}) {
  const router = useRouter();
  const id = useId();
  const [quantidade, setQuantidade] = useState(quantidadeInicial);
  // undefined enquanto a Sessão não foi lida: o botão espera, em vez de mandar
  // ao Login quem já entrou.
  const [comSessao, setComSessao] = useState<boolean | undefined>(undefined);
  const [enviando, setEnviando] = useState(false);
  const [aviso, setAviso] = useState<{ texto: string; erro: boolean } | null>(null);
  // Contra o efeito duplo do StrictMode: a volta do Login adiciona uma vez só.
  const jaAdicionou = useRef(false);

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

  function irAoLogin(q: number) {
    router.push(paraLogin(`/produtos/${produtoId}?quantidade=${q}&adicionar=1`));
  }

  // Devolve false quando levou ao Login: o botão continua desabilitado durante
  // a navegação. A mensagem de recusa é sempre a do envelope.
  async function enviar(q: number): Promise<boolean> {
    setEnviando(true);
    setAviso(null);
    try {
      const { resposta, json } = await pedir("/api/v1/carrinho/itens", "POST", {
        produto_id: produtoId,
        quantidade: q,
      });
      if (resposta.status === 401) {
        irAoLogin(q);
        return false;
      }
      // A recusa por Estoque (4.2) chega com o disponível na própria mensagem:
      // "Restam 3 unidades de …". O contador da barra só muda quando deu certo.
      if (resposta.ok) avisarCarrinhoAlterado();
      setAviso(
        resposta.ok
          ? { texto: "Adicionado ao Carrinho.", erro: false }
          : { texto: json?.erro?.mensagem ?? "Não foi possível adicionar ao Carrinho.", erro: true },
      );
    } catch {
      setAviso({ texto: FALHA_DE_REDE, erro: true });
    }
    setEnviando(false);
    return true;
  }

  useEffect(() => {
    if (!adicionarAoEntrar || comSessao !== true || jaAdicionou.current) return;
    jaAdicionou.current = true;
    // Esgotado na volta: não cria Item nenhum — o ramo "Indisponível" não tem
    // onde dizer que criou. Só tira o marcador da URL.
    if (disponivel === 0) {
      router.replace(`/produtos/${produtoId}`);
      return;
    }
    enviar(quantidadeInicial).then((ficou) => ficou && router.replace(`/produtos/${produtoId}`));
    // Só a Sessão lida dispara: a quantidade da volta é a da URL, e não a do seletor.
  }, [adicionarAoEntrar, comSessao]);

  function adicionar() {
    if (comSessao === false) irAoLogin(quantidade);
    else void enviar(quantidade);
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
              <Button
                size="lg"
                className="w-full rounded-full"
                onClick={adicionar}
                disabled={comSessao === undefined || enviando}
              >
                Adicionar ao Carrinho
              </Button>
              {aviso && (
                <p
                  className={aviso.erro ? "text-destructive text-sm" : "text-sm"}
                  role={aviso.erro ? "alert" : "status"}
                >
                  {aviso.texto}
                </p>
              )}
            </div>
            {children}
          </>
        )}
      </CardContent>
    </Card>
  );
}
