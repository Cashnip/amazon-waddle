"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import {
  comPrecosConfirmados,
  revalidacaoDe,
  textoDoBloqueio,
  type Carrinho,
  type PrecosConfirmados,
} from "@/lib/carrinho";
import {
  aberturaDoPassoEndereco,
  enderecoMarcado,
  ENTRADA_NO_CHECKOUT,
  guardarEscolha,
  lerEscolha,
  podeContinuar,
  razoesDoContinuar,
} from "@/lib/checkout";
import { paraLogin } from "@/lib/destino";
import type { Endereco } from "@/lib/endereco";
import { FALHA_DE_REDE, pedir } from "@/lib/pedir";
import { formatarPreco } from "@/lib/preco";
import { EtapasDoCheckout } from "../etapas";

// O passo Endereço do checkout (FR-20). Nenhuma regra mora aqui (AD-10): a
// lista e o cadastro são as rotas de `/api/v1/enderecos` que "Meus endereços"
// já usa, e a mensagem exibida é a do envelope.
//
// O navegador guarda só o `id` escolhido, e só quando o Comprador clica
// "Continuar". Frete, CEP e total nunca ficam aqui: a Revisão pergunta ao Go.
// Editar e remover Endereço não acontece dentro do checkout — é "Meus
// endereços".
//
// A abertura é `POST /api/v1/checkout/entrada` (5.5), e não a leitura do
// Carrinho: numa transação só, o Go revalida, **reporta** o que mudou e só
// então grava a ciência dos preços reportados (AD-17). É por isso que o aviso
// de preço aparece uma vez e não volta na próxima entrada — e é o que fecha o
// caminho de quem digita a URL por cima do "Fechar o Pedido" bloqueado.
//
// Os dois `Alert` são os do Carrinho, sobre a mesma `revalidacaoDe`: nenhuma
// regra nova mora aqui (AD-10). O que muda é a ação do bloqueio — aqui não há
// Ajustar nem Remover, que são do Carrinho, e sim o caminho de volta a ele.

export function EscolhaDeEndereco() {
  const router = useRouter();
  const [enderecos, setEnderecos] = useState<Endereco[] | null>(null);
  const [marcado, setMarcado] = useState<string | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [cadastrando, setCadastrando] = useState(false);
  // O Carrinho como a entrada no checkout o reportou, e os preços que o
  // Comprador confirmou nesta tela. A ciência já está no banco: a confirmação
  // daqui é o reconhecimento do aviso, como no Carrinho (4.4).
  const [carrinho, setCarrinho] = useState<Carrinho | null>(null);
  const [confirmados, setConfirmados] = useState<PrecosConfirmados>({});
  const titulo = useRef<HTMLHeadingElement>(null);

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

  // A entrada no checkout **escreve** (AD-17), e por isso este efeito não se
  // parece com os outros da casca em dois pontos.
  //
  // Primeiro, ela dispara **uma vez por tentativa**: o StrictMode do `npm run
  // dev` monta, desmonta e remonta, e sem esta trava o primeiro POST gravaria
  // a ciência e o segundo já voltaria silencioso — o "de X para Y" nunca
  // apareceria em desenvolvimento. O `useRef` atravessa as duas montagens.
  //
  // Segundo, **não há bandeira `vivo`**: descartar a resposta de uma entrada
  // que já comitou é perder o relatório para sempre, que é exatamente o que o
  // AD-17 existe para impedir. Em React 19 um `setState` depois da
  // desmontagem é no-op silencioso, e é o preço certo a pagar aqui.
  //
  // Quem decide o que fazer com as duas respostas é `aberturaDoPassoEndereco`,
  // pura e testada: a ordem das guardas — e o fato de o relatório ser aplicado
  // **antes** da guarda da lista de Endereços — mora lá, onde um teste a
  // prende. Cadastrar Endereço depois disso recarrega só a lista, porque uma
  // segunda entrada apagaria o aviso antes de ele ser lido.
  const jaEntrou = useRef(false);
  useEffect(() => {
    if (jaEntrou.current) return;
    jaEntrou.current = true;
    (async () => {
      try {
        const [entrada, lista] = await Promise.all([
          pedir(ENTRADA_NO_CHECKOUT.rota, ENTRADA_NO_CHECKOUT.metodo),
          pedir("/api/v1/enderecos", "GET"),
        ]);
        const abertura = aberturaDoPassoEndereco<Endereco>(entrada, lista);
        if (abertura.semSessao) return semSessao();
        // O relatório primeiro, sempre que veio: o erro e o desvio convivem
        // com os dois `Alert`, e nunca os apagam.
        if (abertura.carrinho !== null) setCarrinho(abertura.carrinho);
        if (abertura.erro !== null) setErro(abertura.erro);
        if (abertura.destino !== null) {
          router.replace(abertura.destino);
          return;
        }
        if (abertura.enderecos !== null) aplicar(abertura.enderecos, null);
      } catch {
        setErro(FALHA_DE_REDE);
      }
    })();
  }, [aplicar, router, semSessao]);

  // A mesma revalidação do Carrinho, sobre o que a entrada reportou. As duas
  // listas são independentes: confirmar preço não desbloqueia. Fica **acima**
  // de `continuar`, e não depois: lida no corpo do componente, uma `const`
  // declarada abaixo estouraria na zona morta.
  const { bloqueios, mudancas, podeAvancar } =
    carrinho === null
      ? { bloqueios: [], mudancas: [], podeAvancar: false }
      : revalidacaoDe(carrinho, confirmados);
  const liberado = podeContinuar(marcado, podeAvancar);

  function continuar() {
    // A guarda é de verdade, e não decoração: o botão usa `aria-disabled`, que
    // não impede o clique — é aqui que o avanço bloqueado para. O
    // `marcado === null` é redundante para quem lê `podeContinuar`, e não para
    // o compilador: é ele que estreita o tipo até o `guardarEscolha` abaixo.
    if (!liberado || marcado === null) return;
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

  // Por que o Continuar está travado. Fica escrito ao lado do botão, e é a
  // descrição acessível dele: um botão morto sem explicação não serve.
  //
  // `!marcado` não entra na lista, e não é esquecimento: o botão só é
  // renderizado com a lista não vazia, e `enderecoMarcado` sempre marca alguém
  // numa lista não vazia — a condição existe como guarda, e não como estado
  // que o Comprador possa ver e não entender.
  const razoes = razoesDoContinuar(bloqueios.length, mudancas.length);
  // O botão e a linha da razão nascem e morrem juntos: um `id` apontando para
  // parte nenhuma é pior que descrição nenhuma.
  const mostraContinuar = enderecos !== null && enderecos.length > 0;

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
      <h1 ref={titulo} tabIndex={-1} className="text-2xl font-medium outline-none">Endereço de entrega</h1>

      {erro && (
        <Alert variant="destructive">
          <AlertDescription>{erro}</AlertDescription>
        </Alert>
      )}

      {/* Trava o avanço até o Produto ser removido ou a quantidade ajustada
          (FR-19). Ajustar e Remover são do Carrinho, e o caminho até lá é a
          ação daqui — editar o Carrinho dentro do checkout seria outra tela. */}
      {bloqueios.length > 0 && (
        <Alert variant="destructive">
          <AlertTitle>Ajuste o Carrinho antes de fazer o Pedido.</AlertTitle>
          <AlertDescription>
            <ul className="space-y-1">
              {bloqueios.map((linha) => (
                <li key={linha.id}>{textoDoBloqueio(linha)}</li>
              ))}
            </ul>
            <Button variant="outline" size="sm" className="mt-3" asChild>
              <a href="/carrinho">Voltar ao Carrinho</a>
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {/* O preço anterior e o atual, com confirmação. A ciência já ficou no
          banco na entrada (AD-17): confirmar aqui é reconhecer o aviso, e por
          isso ele não volta na próxima entrada. */}
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
              onClick={() => {
                setConfirmados((atuais) => comPrecosConfirmados(atuais, mudancas));
                // O `Alert` sai com o próprio botão, e o foco cairia no `body`.
                titulo.current?.focus();
              }}
            >
              {mudancas.length === 1 ? "Confirmar o novo preço" : "Confirmar os novos preços"}
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {conteudo}

      <div className="space-y-2">
        <div className="flex flex-wrap gap-2">
          {/* `aria-disabled`, e não o `disabled` nativo: um botão desabilitado
              de verdade não recebe foco, e o leitor de tela nunca chegaria à
              razão que o `aria-describedby` aponta. O clique é barrado por
              `continuar`, que confere `liberado` antes de qualquer coisa. */}
          {mostraContinuar && (
            <Button
              onClick={continuar}
              aria-disabled={!liberado}
              aria-describedby={razoes.length > 0 ? "razao-continuar" : undefined}
              className="aria-disabled:cursor-not-allowed aria-disabled:opacity-50"
            >
              Continuar
            </Button>
          )}
          <Button variant="outline" asChild>
            <a href="/carrinho">Voltar ao Carrinho</a>
          </Button>
        </div>
        {mostraContinuar && razoes.length > 0 && (
          <p id="razao-continuar" className="text-muted-foreground text-sm">
            {razoes.join(" ")}
          </p>
        )}
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
