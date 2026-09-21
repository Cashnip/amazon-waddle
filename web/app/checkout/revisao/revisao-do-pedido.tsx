"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { ImagemDoProduto } from "@/components/imagem-do-produto";
import { EnderecoPorExtenso } from "@/components/formulario-de-endereco";
import {
  avisarCarrinhoAlterado,
  parcelaDe,
  unidadesTexto,
  type Carrinho,
  type LinhaDoCarrinho,
} from "@/lib/carrinho";
import {
  aberturaDaRevisao,
  chaveDeIdempotencia,
  descartarChave,
  descartarEscolha,
  desfechoDaConfirmacao,
  enderecoEscolhido,
  executarDesfecho,
  freteGratis,
  lerEscolha,
  limparCheckout,
  PRAZO_DA_CONFIRMACAO_MS,
  requisicaoDeConfirmar,
  rotaDoFrete,
  type AvisoDaConfirmacao,
  type Desfecho,
  type Cotacao,
} from "@/lib/checkout";
import { paraLogin } from "@/lib/destino";
import type { Endereco } from "@/lib/endereco";
import { FALHA_DE_REDE, pedir } from "@/lib/pedir";
import { formatarPreco } from "@/lib/preco";
import { EtapasDoCheckout } from "../etapas";

// A Revisão do Pedido (FR-22): o último ponto de leitura antes do compromisso.
// O que ela lista são **Itens de Carrinho** — o Pedido ainda não existe, e só a
// criação congela o preço praticado em Itens de Pedido (PRD §3, FR-23).
// Nenhuma regra mora aqui (AD-10) — os Itens de Carrinho, o subtotal, o Frete e
// o total vêm prontos do Go, e a tela **não soma o total** (NFR-13): ele é o
// `total_centavos` da cotação. A parcela de cada linha é a mesma multiplicação
// que o Carrinho já exibe, e é só apresentação.
//
// A cotação é perguntada a **cada abertura**, e nada dela é guardado no
// navegador — nem Frete, nem CEP, nem total. É isso que faz trocar o Endereço
// recalcular o Frete (FR-20/FR-22). O navegador guarda só o `id` do Endereço
// escolhido (5.2) e a chave de idempotência desta tentativa de checkout.
//
// A chave nasce ao **entrar** na Revisão, e não no clique: gerada no clique, o
// duplo clique produziria duas chaves e dois Pedidos. O Confirmar Pedido a
// envia no `POST /api/v1/pedidos`, com o Endereço e o total que a Revisão
// exibiu, e `limparCheckout` só roda quando o Pedido existe (5.6). O que fazer
// com cada resposta é decisão pura de `desfechoDaConfirmacao`; aqui ela só é
// executada.
//
// A revalidação de Estoque e preço é da entrada do checkout, no passo Endereço
// (5.5). A Revisão **não repete a escrita**: ela desvia, por leitura pura. O
// `GET /api/v1/carrinho` que ela já pedia traz `bloqueio` e `preco_mudou`, e
// `desvioDaRevisao` decide — Carrinho com bloqueio volta ao Carrinho, preço
// que mudou depois da entrada volta ao passo Endereço, que é quem reporta e
// confirma (AD-17). Duas escritas seriam dois lugares gravando o mesmo campo.

// A criação do Pedido a partir do Carrinho existe desde a 5.6, que virou esta
// linha. A guarda de armazenamento abaixo não depende dela e continua de pé
// sozinha.
const CRIACAO_DISPONIVEL: boolean = true;

// As razões de o Confirmar Pedido estar indisponível, em ordem de exibição.
// Ficam numa linha ao lado do botão, e não escondidas num `title`: um botão
// morto sem explicação é o que a UX-DR7 não admite no passo irreversível. A
// mesma linha é a descrição acessível do botão, por `aria-describedby`, então
// o que se lê e o que se ouve nunca divergem.
const RAZAO_SEM_CRIACAO = "A criação do Pedido chega na próxima etapa do projeto.";
const RAZAO_SEM_CHAVE = "Sem a proteção contra confirmação repetida, o Pedido não pode ser confirmado.";
const SEM_ARMAZENAMENTO =
  "Não foi possível guardar a proteção contra confirmação repetida neste navegador. A Revisão pode ser lida, mas confirmar o Pedido fica indisponível.";

export function RevisaoDoPedido() {
  const router = useRouter();
  const [carrinho, setCarrinho] = useState<Carrinho | null>(null);
  const [endereco, setEndereco] = useState<Endereco | null>(null);
  const [cotacao, setCotacao] = useState<Cotacao | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  // `null` quando o `sessionStorage` não respondeu: sem chave, a confirmação
  // não estaria protegida, e a tela diz isso em vez de prometer o contrário.
  // `undefined` é só o instante antes de o efeito rodar.
  const [chave, setChave] = useState<string | null | undefined>(undefined);
  const comChave = chave !== null;
  // O aviso de uma recusa da confirmação. Fica fora de `erro`, que é o de não
  // conseguir abrir a Revisão: o total divergente avisa **e** relê.
  const [aviso, setAviso] = useState<AvisoDaConfirmacao | null>(null);
  // Envio único: o `ref` barra o segundo disparo **no mesmo tique** — o
  // `disabled` do estado só chega no render seguinte, e o duplo clique cabe
  // entre os dois. O estado é o que a tela mostra.
  const emVoo = useRef(false);
  const [enviando, setEnviando] = useState(false);
  // Muda para reler a Revisão inteira depois do `TOTAL_DIVERGENTE`.
  const [leitura, setLeitura] = useState(0);

  useEffect(() => {
    let vivo = true;

    // A chave nasce aqui, na entrada da Revisão — antes de qualquer leitura, e
    // uma só por tentativa de checkout: se já existe (recarregamento, ida e
    // volta ao Carrinho, releitura depois de uma recusa), é a mesma que
    // continua.
    setChave(chaveDeIdempotencia());

    const guardado = lerEscolha();
    if (guardado === null) {
      router.replace("/checkout/endereco");
      return;
    }

    (async () => {
      try {
        // O Carrinho e os Endereços juntos, como no passo Endereço. Carrinho
        // vazio não tem Pedido a fechar, e volta ao Carrinho sem que a Revisão
        // chegue a aparecer — nada é exibido antes das três leituras.
        const [oCarrinho, aLista] = await Promise.all([
          pedir("/api/v1/carrinho", "GET"),
          pedir("/api/v1/enderecos", "GET"),
        ]);
        if (!vivo) return;
        // A decisão inteira desta etapa mora em `aberturaDaRevisao`, pura e
        // testada: 401, Carrinho que não abriu, Carrinho vazio, desvio da 5.5
        // e só então a lista de Endereços. A ordem é o que importa — o desvio
        // vem **antes** de qualquer pintura, e quem digitou esta URL por cima
        // de um avanço bloqueado não vê a Revisão piscar — e no efeito ela não
        // teria como ser presa por teste.
        const abertura = aberturaDaRevisao<Endereco>(oCarrinho, aLista);
        if (abertura.semSessao) {
          router.push(paraLogin());
          return;
        }
        if (abertura.destino !== null) {
          router.replace(abertura.destino);
          return;
        }
        if (abertura.erro !== null || abertura.carrinho === null || abertura.enderecos === null) {
          setErro(abertura.erro ?? "Não foi possível abrir o Carrinho.");
          return;
        }
        const revalidado = abertura.carrinho;

        // Sem escolha válida — nada guardado, ou um `id` removido em "Meus
        // endereços" entre uma visita e outra — o Comprador não escolheu nada,
        // e o passo volta ao Endereço.
        const escolhido = enderecoEscolhido<Endereco>(abertura.enderecos, guardado);
        if (!escolhido) {
          router.replace("/checkout/endereco");
          return;
        }

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

        setCarrinho(revalidado);
        setEndereco(escolhido);
        setCotacao(frete.json);
      } catch {
        if (vivo) setErro(FALHA_DE_REDE);
      }
    })();

    return () => {
      vivo = false;
    };
    // `leitura` não é lida dentro do efeito: mudar de valor é o pedido de
    // reler a Revisão inteira do Go.
  }, [router, leitura]);

  async function confirmar() {
    // A guarda mora aqui, e não no `disabled` nativo: o botão usa
    // `aria-disabled` para continuar focalizável, e a razão, alcançável pelo
    // leitor de tela (o mesmo da 5.5 no Continuar).
    if (razoes.length > 0 || emVoo.current || !chave || endereco === null || cotacao === null) return;
    emVoo.current = true;
    setEnviando(true);
    setAviso(null);
    // Uma requisição pendurada não trava o botão para sempre: esgotado o
    // prazo, a guarda é solta e a mesma chave torna o novo clique seguro.
    const cancelar = new AbortController();
    const prazo = setTimeout(() => cancelar.abort(), PRAZO_DA_CONFIRMACAO_MS);
    let desfecho: Desfecho;
    try {
      const { rota, init } = requisicaoDeConfirmar(chave, endereco.id, cotacao.total_centavos, cancelar.signal);
      const resposta = await fetch(rota, init);
      const json = await resposta.json().catch(() => null);
      desfecho = desfechoDaConfirmacao({ resposta, json });
    } catch {
      desfecho = { tipo: "erro", mensagem: FALHA_DE_REDE, comCarrinho: false };
    } finally {
      clearTimeout(prazo);
    }
    // O que fazer com a resposta é de `executarDesfecho`, pura e testada; aqui
    // só se entregam os efeitos.
    const soltar = executarDesfecho(desfecho, {
      limpar: () => limparCheckout(),
      descartarChave: () => descartarChave(),
      descartarEscolha: () => descartarEscolha(),
      avisarCarrinho: avisarCarrinhoAlterado,
      navegar: (destino) => router.push(destino),
      irAoLogin: () => router.push(paraLogin()),
      reler: () => {
        setCotacao(null);
        setLeitura((n) => n + 1);
      },
      mostrarAviso: setAviso,
    });
    if (soltar) {
      emVoo.current = false;
      setEnviando(false);
    }
  }

  const pronta = carrinho !== null && endereco !== null && cotacao !== null;

  // As razões acumulam: quando as duas valem, as duas são ditas. A lista vazia
  // é o único caso em que o botão fica habilitado — sem armazenamento,
  // `comChave` sozinho mantém o botão travado. O envio em voo também trava,
  // mas não é razão a explicar: o próprio botão diz "Confirmando…".
  const razoes = [
    ...(CRIACAO_DISPONIVEL ? [] : [RAZAO_SEM_CRIACAO]),
    ...(comChave ? [] : [RAZAO_SEM_CHAVE]),
  ];

  return (
    <div className="space-y-4">
      <EtapasDoCheckout atual="Revisão" />
      <h1 className="text-2xl font-medium">Revisão do Pedido</h1>

      {erro && (
        <Alert variant="destructive">
          <AlertDescription>{erro}</AlertDescription>
        </Alert>
      )}

      {/* A recusa da confirmação. O `Alert` já é `role="alert"`: o foco
          continua no botão, e quem usa leitor de tela precisa ouvir por que
          nada aconteceu. */}
      {aviso && (
        <Alert variant="destructive">
          <AlertDescription>
            <p>{aviso.mensagem}</p>
            {aviso.pedido !== null && (
              <a className="text-link hover:underline" href={aviso.pedido}>
                Ver o Pedido
              </a>
            )}
            {aviso.comCarrinho && (
              <a className="text-link hover:underline" href="/carrinho">
                Voltar ao Carrinho
              </a>
            )}
          </AlertDescription>
        </Alert>
      )}

      {!pronta && !erro && (
        <p className="text-muted-foreground text-sm" role="status">
          Carregando a Revisão…
        </p>
      )}

      {carrinho !== null && endereco !== null && cotacao !== null && (
        <>
          <Card>
            <CardHeader>
              <CardTitle>
                <h2>Itens do Carrinho</h2>
              </CardTitle>
            </CardHeader>
            <CardContent>
              {/* Só leitura: quantidade, remoção e preço são do Carrinho. */}
              <ul>
                {carrinho.itens.map((linha, i) => (
                  <li key={linha.id}>
                    {i > 0 && <Separator className="my-4" />}
                    <LinhaDaRevisao linha={linha} />
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>

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

          {/* O bloco de valores é o do Go inteiro. Quando o subtotal da cotação
              discorda do que o Carrinho devolveu, porque um preço mudou entre
              as duas leituras, a Revisão não arbitra: quem recusa é a criação
              do Pedido, sob a trava, com `TOTAL_DIVERGENTE` (5.6). */}
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
                  <dd className="tabular-nums">
                    {freteGratis(cotacao) ? "Grátis" : formatarPreco(cotacao.frete_centavos)}
                  </dd>
                </div>
                <div className="flex justify-between gap-4 border-t pt-2 font-medium">
                  <dt>Total</dt>
                  <dd className="tabular-nums">{formatarPreco(cotacao.total_centavos)}</dd>
                </div>
              </dl>
            </CardContent>
          </Card>

          {/* A declaração de pagamento é uma frase, e nada mais: nenhum campo,
              rótulo, máscara ou *placeholder* de cartão existe em lugar nenhum
              do sistema (PRD §8). */}
          <Card>
            <CardHeader>
              <CardTitle>
                <h2>Pagamento</h2>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm">
                Ao confirmar, o Pedido é enviado ao Provedor de Pagamento. Nenhum dado de cartão é pedido nem
                guardado, e o resultado chega depois, na tela do Pedido.
              </p>
            </CardContent>
          </Card>

          {!comChave && (
            <Alert>
              <AlertDescription>{SEM_ARMAZENAMENTO}</AlertDescription>
            </Alert>
          )}

          <div className="flex flex-col items-end gap-2">
            {/* O passo irreversível, e a **única** aparição do laranja no fluxo
                inteiro (UX-DR7, DESIGN.md): `{colors.primary-strong}` em pill.
                `aria-disabled`, e não o `disabled` nativo: botão desabilitado
                de verdade não recebe foco, e a razão ligada por
                `aria-describedby` nunca chegaria ao leitor de tela (o mesmo da
                5.5 no Continuar). Quem barra o envio é a guarda dentro de
                `confirmar`; a trava deriva das razões e do envio em voo, e a
                guarda de armazenamento trava o botão por conta própria. */}
            <Button
              size="lg"
              className="rounded-full bg-primary-strong text-primary-strong-foreground hover:bg-primary-strong/90 aria-disabled:cursor-not-allowed aria-disabled:opacity-50"
              aria-disabled={razoes.length > 0 || enviando}
              aria-describedby={razoes.length > 0 ? "razao-confirmar" : undefined}
              onClick={confirmar}
            >
              {enviando ? "Confirmando…" : "Confirmar Pedido"}
            </Button>
            {razoes.length > 0 && (
              <p id="razao-confirmar" className="text-muted-foreground text-right text-sm">
                {razoes.join(" ")}
              </p>
            )}
          </div>
        </>
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

// Uma linha de Item de Carrinho, no molde da do Carrinho e sem as ações dele: preço
// unitário à esquerda, parcela à direita. O Produto que saiu da visibilidade
// não tem nome nem preço — quem o resolve é o Carrinho, e a revalidação na
// entrada do checkout é da 5.5.
function LinhaDaRevisao({ linha }: { linha: LinhaDoCarrinho }) {
  if (!linha.visivel) {
    return <p className="text-muted-foreground">Produto indisponível.</p>;
  }
  return (
    <div className="grid grid-cols-[4rem_minmax(0,1fr)] gap-4 sm:grid-cols-[6rem_minmax(0,1fr)_auto]">
      <ImagemDoProduto src={linha.imagem_url} nome={linha.nome} compacta />
      <div className="min-w-0 space-y-1">
        <a className="text-link break-words hover:underline" href={`/produtos/${linha.produto_id}`}>
          {linha.nome}
        </a>
        <p className="text-muted-foreground text-sm tabular-nums">
          {formatarPreco(linha.preco_centavos)} cada · {unidadesTexto(linha.quantidade)}
        </p>
      </div>
      <p className="col-span-2 text-right font-medium tabular-nums sm:col-span-1">{formatarPreco(parcelaDe(linha))}</p>
    </div>
  );
}
