"use client";

import { useCallback, useEffect, useId, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
  comPrecosConfirmados,
  comQuantidade,
  faltaParaFreteGratis,
  parcelaDe,
  quantidadeDoCampo,
  revalidacaoDe,
  textoDoBloqueio,
  unidadesTexto,
  type Carrinho,
  type LinhaDoCarrinho,
  type PrecosConfirmados,
} from "@/lib/carrinho";
import { iniciarTentativaDeCheckout } from "@/lib/checkout";
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
//
// A revalidação da FR-19 acontece na abertura: o `GET` já diz, por linha, se o
// preço mudou e o que bloqueia (decidido no Go). Os dois `Alert` só mostram. O
// que o Comprador confirma fica aqui, na tela — nada é gravado (AD-17), e por
// isso o aviso de preço volta quando o Carrinho é reaberto.

export function MeuCarrinho() {
  const router = useRouter();

  // "Fechar o Pedido" começa uma tentativa de checkout: a chave de
  // idempotência da anterior sai, e a Revisão gera outra ao abrir. A escolha
  // de Endereço fica (FR-22). Ver `iniciarTentativaDeCheckout`.
  function fecharOPedido() {
    iniciarTentativaDeCheckout();
    router.push("/checkout/endereco");
  }
  const [carrinho, setCarrinho] = useState<Carrinho | null>(null);
  const [erroDaLista, setErroDaLista] = useState<string | null>(null);
  // A mensagem de uma edição ou remoção que falhou, por linha.
  const [avisos, setAvisos] = useState<Record<string, string>>({});
  const [confirmados, setConfirmados] = useState<PrecosConfirmados>({});
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

  const { bloqueios, mudancas, podeAvancar } = revalidacaoDe(carrinho, confirmados);
  const falta = faltaParaFreteGratis(carrinho.subtotal_centavos, carrinho.frete_isencao_centavos);

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-medium">Carrinho</h1>
      {erroDaLista && (
        <Alert variant="destructive">
          <AlertDescription>{erroDaLista}</AlertDescription>
        </Alert>
      )}

      {/* Bloqueia o Pedido até o Produto ser removido ou a quantidade ajustada,
          com as duas ações aqui dentro (FR-19). Confirmar o preço não o tira. */}
      {bloqueios.length > 0 && (
        <Alert variant="destructive">
          <AlertTitle>Ajuste o Carrinho antes de fazer o Pedido.</AlertTitle>
          <AlertDescription>
            <ul className="space-y-3">
              {bloqueios.map((linha) => (
                <li key={linha.id} className="flex flex-wrap items-center justify-between gap-2">
                  <span>{textoDoBloqueio(linha)}</span>
                  <span className="flex flex-wrap gap-2">
                    {linha.bloqueio === "acima_do_estoque" && (
                      <Button size="sm" variant="outline" onClick={() => alterar(linha, linha.estoque_disponivel)}>
                        Ajustar para {unidadesTexto(linha.estoque_disponivel)}
                      </Button>
                    )}
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => remover(linha)}
                      aria-label={linha.visivel ? `Remover ${linha.nome} do Carrinho` : "Remover o Produto indisponível do Carrinho"}
                    >
                      Remover
                    </Button>
                  </span>
                  {avisos[linha.id] && <span className="w-full text-sm">{avisos[linha.id]}</span>}
                </li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      )}

      {/* O preço anterior e o atual de cada Produto, com confirmação. A ciência
          só fica na tela: quem a grava é o checkout (AD-17). */}
      {mudancas.length > 0 && (
        <Alert>
          <AlertTitle>{mudancas.length === 1 ? "O preço mudou." : "Os preços mudaram."}</AlertTitle>
          <AlertDescription>
            <ul className="space-y-1">
              {mudancas.map((linha) => (
                <li key={linha.id}>
                  {linha.nome}: de{" "}
                  <span className="tabular-nums">{formatarPreco(linha.preco_visto_centavos)}</span> para{" "}
                  <span className="font-medium tabular-nums">{formatarPreco(linha.preco_centavos)}</span>.
                </li>
              ))}
            </ul>
            <Button
              size="sm"
              variant="outline"
              className="mt-3"
              onClick={() => setConfirmados((atuais) => comPrecosConfirmados(atuais, mudancas))}
            >
              {mudancas.length === 1 ? "Confirmar o novo preço" : "Confirmar os novos preços"}
            </Button>
          </AlertDescription>
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
        {/* Subtotal e nada mais (UX-DR17): o Frete depende do CEP e só aparece
            no checkout. A distância até a isenção é uma frase em valor
            absoluto, sem barra de progresso, e cala quando já foi alcançada. */}
        <div className="text-right">
          <p className="text-lg">
            Subtotal: <span className="font-medium tabular-nums">{formatarPreco(carrinho.subtotal_centavos)}</span>
          </p>
          {falta > 0 && (
            <p className="text-muted-foreground text-sm">
              Faltam <span className="tabular-nums">{formatarPreco(falta)}</span> para o Frete grátis.
            </p>
          )}
          {/* A entrada do checkout (5.2). Desabilitado enquanto há bloqueio ou
              preço a confirmar — os dois `Alert` acima dizem o quê. Não é o
              passo irreversível, por isso sem laranja: esse é o Confirmar
              Pedido, na Revisão. */}
          <Button className="mt-3" disabled={!podeAvancar} onClick={fecharOPedido}>
            Fechar o Pedido
          </Button>
        </div>
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
// visibilidade não tem nome nem preço: a linha só diz que está indisponível, e
// o remover está no `Alert` de bloqueio (4.4). A mensagem de uma ação que falhou
// numa linha bloqueada também aparece lá, e não repetida aqui.
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
    return <p className="text-muted-foreground">Produto indisponível.</p>;
  }
  const avisoNaLinha = linha.bloqueio === "" ? aviso : undefined;

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
            aria-describedby={avisoNaLinha ? idAviso : undefined}
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
        {avisoNaLinha && (
          <p id={idAviso} className="text-destructive text-sm" role="alert">
            {avisoNaLinha}
          </p>
        )}
      </div>
      <p className="col-span-2 text-right font-medium tabular-nums sm:col-span-1">{formatarPreco(parcelaDe(linha))}</p>
    </div>
  );
}
