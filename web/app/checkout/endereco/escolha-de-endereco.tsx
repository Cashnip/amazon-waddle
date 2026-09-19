"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { FormularioDeEndereco, LinhasDoEndereco } from "@/components/formulario-de-endereco";
import type { Carrinho } from "@/lib/carrinho";
import { enderecoMarcado, guardarEscolha, lerEscolha } from "@/lib/checkout";
import { paraLogin } from "@/lib/destino";
import type { Endereco } from "@/lib/endereco";
import { FALHA_DE_REDE, pedir } from "@/lib/pedir";
import { EtapasDoCheckout } from "../etapas";

// O passo Endereço do checkout (FR-20). Nenhuma regra mora aqui (AD-10): a
// lista e o cadastro são as rotas de `/api/v1/enderecos` que "Meus endereços"
// já usa, e a mensagem exibida é a do envelope.
//
// O navegador guarda só o `id` escolhido, e só quando o Comprador clica
// "Continuar". Frete, CEP e total nunca ficam aqui: a Revisão pergunta ao Go.
// Editar e remover Endereço não acontece dentro do checkout — é "Meus
// endereços".

export function EscolhaDeEndereco() {
  const router = useRouter();
  const [enderecos, setEnderecos] = useState<Endereco[] | null>(null);
  const [marcado, setMarcado] = useState<string | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [cadastrando, setCadastrando] = useState(false);

  // O 401 leva ao Login com o passo atual no `destino`.
  const semSessao = useCallback(() => router.push(paraLogin()), [router]);

  // A lista, e o marcado refeito sobre ela: o `preferido` (o Endereço recém-
  // cadastrado) se veio, senão o guardado; e, sem nenhum dos dois na lista, o
  // primeiro.
  const aplicar = useCallback((lista: Endereco[], preferido: string | null) => {
    setEnderecos(lista);
    setMarcado(enderecoMarcado(lista, lerEscolha(), preferido));
  }, []);

  const recarregar = useCallback(
    async (preferido: string | null) => {
      try {
        const { resposta, json } = await pedir("/api/v1/enderecos", "GET");
        if (resposta.status === 401) return semSessao();
        if (!resposta.ok) {
          setErro(json?.erro?.mensagem ?? "Não foi possível ler os Endereços.");
          return;
        }
        setErro(null);
        aplicar(json ?? [], preferido);
      } catch {
        setErro(FALHA_DE_REDE);
      }
    },
    [aplicar, semSessao],
  );

  // Na abertura, o Carrinho e os Endereços juntos. Carrinho vazio não tem
  // Pedido a fechar, e o passo devolve o Comprador ao Carrinho.
  useEffect(() => {
    let vivo = true;
    (async () => {
      try {
        const [carrinho, lista] = await Promise.all([
          pedir("/api/v1/carrinho", "GET"),
          pedir("/api/v1/enderecos", "GET"),
        ]);
        if (!vivo) return;
        if (carrinho.resposta.status === 401 || lista.resposta.status === 401) return semSessao();
        if (!carrinho.resposta.ok || !carrinho.json) {
          setErro(carrinho.json?.erro?.mensagem ?? "Não foi possível abrir o Carrinho.");
          return;
        }
        if ((carrinho.json as Carrinho).itens.length === 0) {
          router.replace("/carrinho");
          return;
        }
        if (!lista.resposta.ok) {
          setErro(lista.json?.erro?.mensagem ?? "Não foi possível ler os Endereços.");
          return;
        }
        aplicar(lista.json ?? [], null);
      } catch {
        if (vivo) setErro(FALHA_DE_REDE);
      }
    })();
    return () => {
      vivo = false;
    };
  }, [aplicar, router, semSessao]);

  function continuar() {
    if (!marcado) return;
    // Um aviso de tentativa anterior não fica na tela depois de uma nova.
    setErro(null);
    // Sem armazenamento, a Revisão não teria o que mostrar e devolveria o
    // Comprador para cá sem explicação. Melhor dizer aqui.
    if (!guardarEscolha(marcado)) {
      setErro("Não foi possível guardar a escolha neste navegador.");
      return;
    }
    router.push("/checkout/revisao");
  }

  async function cadastrado(novo: Endereco) {
    setCadastrando(false);
    await recarregar(novo.id);
  }

  let conteudo: React.ReactNode;
  if (enderecos === null) {
    conteudo = erro ? null : (
      <p className="text-muted-foreground text-sm" role="status">
        Carregando os Endereços…
      </p>
    );
  } else if (enderecos.length === 0) {
    // Sem Endereço, o formulário vem aberto direto — uma lista vazia seria um
    // passo a mais sem nada para escolher.
    conteudo = (
      <Card>
        <CardHeader>
          <CardTitle>
            <h2>Novo Endereço</h2>
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-muted-foreground text-sm">Cadastre o Endereço de entrega do Pedido.</p>
          <FormularioDeEndereco endereco={null} aoSalvar={cadastrado} />
        </CardContent>
      </Card>
    );
  } else {
    conteudo = (
      <>
        <RadioGroup
          value={marcado ?? undefined}
          onValueChange={setMarcado}
          aria-label="Endereço de entrega"
          className="gap-3"
        >
          {enderecos.map((endereco) => {
            const id = `endereco-${endereco.id}`;
            return (
              <Label
                key={endereco.id}
                htmlFor={id}
                className="has-[[data-state=checked]]:border-foreground flex cursor-pointer items-start gap-3 rounded-lg border p-4 text-base leading-normal font-normal"
              >
                <RadioGroupItem value={endereco.id} id={id} className="mt-1" />
                <span className="min-w-0 space-y-1 break-words">
                  <LinhasDoEndereco endereco={endereco} />
                </span>
              </Label>
            );
          })}
        </RadioGroup>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" onClick={() => setCadastrando(true)}>
            Cadastrar Endereço
          </Button>
        </div>
      </>
    );
  }

  return (
    <div className="space-y-4">
      <EtapasDoCheckout atual="Endereço" />
      <h1 className="text-2xl font-medium">Endereço de entrega</h1>

      {erro && (
        <Alert variant="destructive">
          <AlertDescription>{erro}</AlertDescription>
        </Alert>
      )}

      {conteudo}

      <div className="flex flex-wrap gap-2">
        {enderecos !== null && enderecos.length > 0 && (
          <Button onClick={continuar} disabled={!marcado}>
            Continuar
          </Button>
        )}
        <Button variant="outline" asChild>
          <a href="/carrinho">Voltar ao Carrinho</a>
        </Button>
      </div>

      {/* `Dialog` de um nível só, e `Esc` fecha: é o comportamento do Radix. O
          409 do teto por Comprador aparece no alerta do próprio formulário. */}
      <Dialog open={cadastrando} onOpenChange={setCadastrando}>
        <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Novo Endereço</DialogTitle>
            <DialogDescription>O Endereço novo fica marcado para este Pedido.</DialogDescription>
          </DialogHeader>
          {cadastrando && (
            <FormularioDeEndereco endereco={null} aoSalvar={cadastrado} aoCancelar={() => setCadastrando(false)} />
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
