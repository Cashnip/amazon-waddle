"use client";

import { useCallback, useEffect, useId, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { ImagemDoProduto } from "@/components/imagem-do-produto";
import {
  avisarCarrinhoAlterado,
  comQuantidade,
  parcelaDe,
  quantidadeDoCampo,
  type Carrinho,
  type LinhaDoCarrinho,
} from "@/lib/carrinho";
import { paraLogin } from "@/lib/destino";
import { FALHA_DE_REDE, pedir } from "@/lib/pedir";
import { formatarPreco } from "@/lib/preco";

// Nenhuma regra mora aqui (AD-10): teto, Estoque e preço são do Go, e a tela
// mostra a mensagem do envelope, sem reescrevê-la. Toda leitura é por ação do
// Comprador — não há consulta em intervalo.
//
// A quantidade é a única coisa com atualização otimista (UX-DR18): a linha e o
// subtotal mudam na hora, e a resposta do servidor ou confirma — a tela
// recarrega o Carrinho e passa a valer o número dele — ou reverte, com a
// mensagem na própria linha. Zero e "Remover" esperam o servidor.

export function MeuCarrinho() {
  const router = useRouter();
  const [carrinho, setCarrinho] = useState<Carrinho | null>(null);
  const [erroDaLista, setErroDaLista] = useState<string | null>(null);
  // A mensagem de uma edição ou remoção que falhou, por linha.
  const [avisos, setAvisos] = useState<Record<string, string>>({});
  const [confirmando, setConfirmando] = useState(false);
  const [esvaziando, setEsvaziando] = useState(false);
  const [erroDoDialog, setErroDoDialog] = useState<string | null>(null);

  const semSessao = useCallback(() => router.push(paraLogin()), [router]);

  // Devolve o Carrinho lido, ou null quando não leu: o 401 já mandou ao Login,
  // e o resto virou `erroDaLista`.
  const carregar = useCallback(async (): Promise<Carrinho | null> => {
    try {
      const { resposta, json } = await pedir("/api/v1/carrinho", "GET");
      if (resposta.status === 401) {
        semSessao();
        return null;
      }
      if (!resposta.ok || !json) {
        setErroDaLista(json?.erro?.mensagem ?? "Não foi possível abrir o Carrinho.");
        return null;
      }
      setErroDaLista(null);
      setCarrinho(json);
      return json;
    } catch {
      setErroDaLista(FALHA_DE_REDE);
      return null;
    }
  }, [semSessao]);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  function avisar(itemId: string, texto: string | null) {
    setAvisos((atuais) => {
      const proximos = { ...atuais };
      if (texto === null) delete proximos[itemId];
      else proximos[itemId] = texto;
      return proximos;
    });
  }

  async function remover(linha: LinhaDoCarrinho) {
    avisar(linha.id, null);
    try {
      const { resposta, json } = await pedir(`/api/v1/carrinho/itens/${linha.id}`, "DELETE");
      if (resposta.status === 401) return semSessao();
      if (!resposta.ok) {
        avisar(linha.id, json?.erro?.mensagem ?? "Não foi possível remover.");
        return;
      }
      await carregar();
      avisarCarrinhoAlterado();
    } catch {
      avisar(linha.id, FALHA_DE_REDE);
    }
  }

  async function alterar(linha: LinhaDoCarrinho, quantidade: number) {
    if (!carrinho) return;
    avisar(linha.id, null);
    if (quantidade === 0) return remover(linha);

    const anterior = carrinho;
    setCarrinho(comQuantidade(anterior, linha.id, quantidade));
    try {
      const { resposta, json } = await pedir(`/api/v1/carrinho/itens/${linha.id}`, "PATCH", { quantidade });
      if (resposta.status === 401) return semSessao();
      if (!resposta.ok) {
        // Reverte na hora, e recarrega em seguida: com duas edições em voo, o
        // retrato de antes de uma já leva a otimista da outra, e só o servidor
        // sabe qual é o Carrinho de verdade.
        setCarrinho(anterior);
        avisar(linha.id, json?.erro?.mensagem ?? "Não foi possível alterar a quantidade.");
        void carregar();
        return;
      }
      await carregar();
      avisarCarrinhoAlterado();
    } catch {
      setCarrinho(anterior);
      avisar(linha.id, FALHA_DE_REDE);
      void carregar();
    }
  }

  async function esvaziar() {
    setEsvaziando(true);
    setErroDoDialog(null);
    try {
      const { resposta, json } = await pedir("/api/v1/carrinho/itens", "DELETE");
      if (resposta.status === 401) return semSessao();
      if (!resposta.ok) {
        setErroDoDialog(json?.erro?.mensagem ?? "Não foi possível esvaziar o Carrinho.");
        return;
      }
      setConfirmando(false);
      await carregar();
      avisarCarrinhoAlterado();
    } catch {
      setErroDoDialog(FALHA_DE_REDE);
    } finally {
      setEsvaziando(false);
    }
  }

  if (erroDaLista && !carrinho) {
    return (
      <Alert variant="destructive">
        <AlertDescription>{erroDaLista}</AlertDescription>
      </Alert>
    );
  }
  if (!carrinho) {
    return (
      <p className="text-muted-foreground text-sm" role="status">
        Carregando o Carrinho…
      </p>
    );
  }

  if (carrinho.itens.length === 0) {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-medium">Carrinho</h1>
        <p>Seu Carrinho está vazio.</p>
        <Button asChild>
          <a href="/">Voltar à Vitrine</a>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-medium">Carrinho</h1>
      {erroDaLista && (
        <Alert variant="destructive">
          <AlertDescription>{erroDaLista}</AlertDescription>
        </Alert>
      )}

      <ul>
        {carrinho.itens.map((linha, i) => (
          <li key={linha.id}>
            {i > 0 && <Separator className="my-4" />}
            <Linha linha={linha} aviso={avisos[linha.id]} aoAlterar={alterar} aoRemover={remover} />
          </li>
        ))}
      </ul>

      <Separator />
      <div className="flex flex-wrap items-center justify-between gap-4">
        <Button variant="outline" onClick={() => setConfirmando(true)}>
          Esvaziar o Carrinho
        </Button>
        <p className="text-lg">
          Subtotal: <span className="font-medium tabular-nums">{formatarPreco(carrinho.subtotal_centavos)}</span>
        </p>
      </div>

      {/* `Dialog` de um nível só, e `Esc` fecha: é o comportamento do Radix. */}
      <Dialog
        open={confirmando}
        onOpenChange={(aberto) => {
          if (esvaziando) return;
          setConfirmando(aberto);
          if (!aberto) setErroDoDialog(null);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Esvaziar o Carrinho</DialogTitle>
            <DialogDescription>Todos os Produtos saem do Carrinho. Esta ação não pode ser desfeita.</DialogDescription>
          </DialogHeader>
          {erroDoDialog && (
            <p className="text-destructive text-sm" role="alert">
              {erroDoDialog}
            </p>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmando(false)} disabled={esvaziando}>
              Cancelar
            </Button>
            <Button onClick={esvaziar} disabled={esvaziando}>
              {esvaziando ? "Esvaziando…" : "Esvaziar"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// Uma linha de Item de Carrinho (UX-DR8): `Input` e `Button`, separada por
// `Separator` pelo pai, sem `Card` aninhado. O Produto que saiu da
// visibilidade não tem nome nem preço: a linha diz que está indisponível e só
// oferece remover. O aviso que bloqueia o avanço é da 4.4.
function Linha({
  linha,
  aviso,
  aoAlterar,
  aoRemover,
}: {
  linha: LinhaDoCarrinho;
  aviso: string | undefined;
  aoAlterar: (linha: LinhaDoCarrinho, quantidade: number) => void;
  aoRemover: (linha: LinhaDoCarrinho) => void;
}) {
  const id = useId();
  const idAviso = `${id}-aviso`;

  if (!linha.visivel) {
    return (
      <div className="flex flex-wrap items-center justify-between gap-4">
        <p className="text-muted-foreground">Produto indisponível.</p>
        <Button variant="outline" onClick={() => aoRemover(linha)} aria-label="Remover o Produto indisponível do Carrinho">
          Remover
        </Button>
        {aviso && (
          <p className="text-destructive w-full text-sm" role="alert">
            {aviso}
          </p>
        )}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-[6rem_minmax(0,1fr)] gap-4 sm:grid-cols-[8rem_minmax(0,1fr)_auto]">
      <ImagemDoProduto src={linha.imagem_url} nome={linha.nome} compacta />
      <div className="min-w-0 space-y-2">
        <a className="text-link break-words hover:underline" href={`/produtos/${linha.produto_id}`}>
          {linha.nome}
        </a>
        <p className="text-muted-foreground text-sm tabular-nums">{formatarPreco(linha.preco_centavos)} cada</p>
        <div className="flex flex-wrap items-center gap-2">
          <Label htmlFor={id}>Quantidade</Label>
          {/* O campo não é controlado, então a `key` é o que o refaz: a
              quantidade que a tela mostra é a que o campo mostra, também depois
              de uma reversão e de um zero que o servidor recusou, que deixaria
              o "0" digitado no campo. */}
          <Input
            key={`${linha.quantidade}|${aviso ?? ""}`}
            id={id}
            className="w-16 tabular-nums"
            inputMode="numeric"
            defaultValue={linha.quantidade}
            aria-describedby={aviso ? idAviso : undefined}
            aria-invalid={aviso ? true : undefined}
            onKeyDown={(e) => {
              if (e.key === "Enter") e.currentTarget.blur();
            }}
            onBlur={(e) => {
              const q = quantidadeDoCampo(e.currentTarget.value);
              if (q === null || q === linha.quantidade) {
                e.currentTarget.value = String(linha.quantidade);
                return;
              }
              aoAlterar(linha, q);
            }}
          />
          <Button variant="outline" onClick={() => aoRemover(linha)} aria-label={`Remover ${linha.nome} do Carrinho`}>
            Remover
          </Button>
        </div>
        {aviso && (
          <p id={idAviso} className="text-destructive text-sm" role="alert">
            {aviso}
          </p>
        )}
      </div>
      <p className="col-span-2 text-right font-medium tabular-nums sm:col-span-1">{formatarPreco(parcelaDe(linha))}</p>
    </div>
  );
}
