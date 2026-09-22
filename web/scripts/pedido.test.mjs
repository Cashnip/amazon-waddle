// A tela do Pedido (5.8): as linhas da matriz que são leitura da resposta do
// Go. O que o Go decide (prazo, terminal, tentativas, disponível) é testado
// lá, em api/pedido_detalhe_test.go.
import { test } from "node:test";
import assert from "node:assert/strict";

const {
  FALHA_NA_NOVA_TENTATIVA,
  INTERVALO_AGUARDANDO_MS,
  INTERVALO_AVANCANDO_MS,
  desfechoDaNovaTentativa,
  impedimentosDaNovaTentativa,
  intervaloDaConsulta,
  motivoDaRecusa,
  podeTentarDeNovo,
  rotaDaNovaTentativa,
  superficieDoPedido,
  tempoRestante,
  textoDasTentativas,
} = await import(new URL("../lib/pedido.ts", import.meta.url).href);

test("cada Status abre a sua superfície; o Detalhe cobre todo o resto", () => {
  assert.equal(superficieDoPedido({ status: "AGUARDANDO_PAGAMENTO" }), "processando");
  assert.equal(superficieDoPedido({ status: "PAGAMENTO_RECUSADO" }), "recusado");
  for (const status of ["PAGO", "SEPARANDO", "ENVIADO", "ENTREGUE", "CANCELADO"]) {
    assert.equal(superficieDoPedido({ status }), "detalhe", status);
  }
});

test("3 s aguardando, 10 s no resto, e para só no terminal do Go", () => {
  assert.equal(intervaloDaConsulta({ status: "AGUARDANDO_PAGAMENTO", terminal: false }), INTERVALO_AGUARDANDO_MS);
  assert.equal(INTERVALO_AGUARDANDO_MS, 3000);
  // Recusado não para: a nova Tentativa pode trazê-lo de volta a aguardando.
  assert.equal(intervaloDaConsulta({ status: "PAGAMENTO_RECUSADO", terminal: false }), INTERVALO_AVANCANDO_MS);
  assert.equal(intervaloDaConsulta({ status: "PAGO", terminal: false }), INTERVALO_AVANCANDO_MS);
  assert.equal(INTERVALO_AVANCANDO_MS, 10000);
  assert.equal(intervaloDaConsulta({ status: "ENTREGUE", terminal: true }), null);
  assert.equal(intervaloDaConsulta({ status: "CANCELADO", terminal: true }), null);
});

test("o tempo restante sai do instante absoluto", () => {
  const expira = "2026-09-22T12:01:00Z";
  const fim = Date.parse(expira);

  assert.deepEqual(tempoRestante(expira, fim - 60_000), { ms: 60_000, texto: "1:00", esgotado: false });
  assert.deepEqual(tempoRestante(expira, fim - 42_000), { ms: 42_000, texto: "0:42", esgotado: false });
  // Arredonda para cima: com 400 ms faltando ainda não é "0:00".
  assert.equal(tempoRestante(expira, fim - 400).texto, "0:01");
  // Quinze minutos, o prazo de produção.
  assert.equal(tempoRestante(expira, fim - 15 * 60_000).texto, "15:00");
});

test("relógio zerado: esgotado, sem tempo negativo", () => {
  const expira = "2026-09-22T12:01:00Z";
  const fim = Date.parse(expira);
  assert.deepEqual(tempoRestante(expira, fim), { ms: 0, texto: "0:00", esgotado: true });
  assert.deepEqual(tempoRestante(expira, fim + 90_000), { ms: 0, texto: "0:00", esgotado: true });
});

test("instante ilegível não vira relógio", () => {
  assert.equal(tempoRestante("amanhã", Date.now()), null);
});

test("o motivo é o da última recusa, com a frase própria da expiração", () => {
  const nascimento = { de: null, para: "AGUARDANDO_PAGAMENTO", ator: "COMPRADOR", motivo: null, em: "t0" };
  const expirou = { de: "AGUARDANDO_PAGAMENTO", para: "PAGAMENTO_RECUSADO", ator: "VARREDURA", motivo: "TEMPO_ESGOTADO", em: "t1" };
  const novaTentativa = { de: "PAGAMENTO_RECUSADO", para: "AGUARDANDO_PAGAMENTO", ator: "COMPRADOR", motivo: null, em: "t2" };
  const recusou = { de: "AGUARDANDO_PAGAMENTO", para: "PAGAMENTO_RECUSADO", ator: "PROVEDOR", motivo: null, em: "t3" };

  assert.equal(motivoDaRecusa([nascimento, expirou]), "Tempo de pagamento expirado.");
  // A segunda recusa é do Provedor: a expiração anterior não pode vencer.
  assert.equal(motivoDaRecusa([nascimento, expirou, novaTentativa, recusou]), "O pagamento foi recusado.");
  assert.equal(motivoDaRecusa([nascimento]), "O pagamento foi recusado.");
});

test("as tentativas restantes, com o termo do glossário", () => {
  assert.equal(textoDasTentativas(2), "Restam 2 Tentativas de Pagamento para este Pedido.");
  assert.equal(textoDasTentativas(1), "Resta 1 Tentativa de Pagamento para este Pedido.");
  assert.equal(textoDasTentativas(0), "Não restam Tentativas de Pagamento para este Pedido.");
});

test("a recusa do Provedor tem frase própria, e o motivo desconhecido cai na genérica", () => {
  const nascimento = { de: null, para: "AGUARDANDO_PAGAMENTO", ator: "COMPRADOR", motivo: null, em: "t0" };
  const recusou = {
    de: "AGUARDANDO_PAGAMENTO",
    para: "PAGAMENTO_RECUSADO",
    ator: "PROVEDOR",
    motivo: "RECUSADO_PELO_PROVEDOR",
    em: "t1",
  };
  assert.equal(motivoDaRecusa([nascimento, recusou]), "O Provedor de Pagamento recusou a Tentativa de Pagamento.");
  assert.equal(motivoDaRecusa([nascimento, { ...recusou, motivo: "OUTRO" }]), "O pagamento foi recusado.");
  // Chave herdada de Object.prototype não é motivo conhecido.
  assert.equal(motivoDaRecusa([nascimento, { ...recusou, motivo: "constructor" }]), "O pagamento foi recusado.");
});

// A tripla do AD-18: Status, Tentativas restantes e disponível por Item.
const item = (nome, disponivel) => ({
  produto_id: nome,
  nome,
  quantidade: 1,
  preco_praticado_centavos: 4990,
  disponivel,
});
const recusado = { status: "PAGAMENTO_RECUSADO", tentativas_restantes: 2, itens: [item("Relógio", true), item("Caneca", true)] };

test("tentar de novo só com recusado, Tentativa restante e todo Item disponível", () => {
  assert.equal(podeTentarDeNovo(recusado), true);
  assert.equal(podeTentarDeNovo({ ...recusado, tentativas_restantes: 0 }), false);
  assert.equal(podeTentarDeNovo({ ...recusado, itens: [item("Relógio", true), item("Caneca", false)] }), false);
  for (const status of ["AGUARDANDO_PAGAMENTO", "PAGO", "CANCELADO"]) {
    assert.equal(podeTentarDeNovo({ ...recusado, status }), false, status);
  }
});

test("sem Estoque, a tela nomeia cada Produto; sem Tentativa, quem fala é o texto das Tentativas", () => {
  assert.deepEqual(impedimentosDaNovaTentativa(recusado), []);
  assert.deepEqual(impedimentosDaNovaTentativa({ ...recusado, itens: [item("Relógio", false), item("Caneca", true)] }), [
    { chave: "Relógio", texto: "Não há Estoque disponível de Relógio para uma nova Tentativa de Pagamento." },
  ]);
  // Dois Itens com o mesmo nome congelado: a chave é o Produto, e não a frase.
  const gemeos = [
    { ...item("Caneca", false), produto_id: "p1" },
    { ...item("Caneca", false), produto_id: "p2" },
  ];
  assert.deepEqual(
    impedimentosDaNovaTentativa({ ...recusado, itens: gemeos }).map((i) => i.chave),
    ["p1", "p2"],
  );
  const semNada = { ...recusado, tentativas_restantes: 0, itens: [item("Relógio", false)] };
  assert.deepEqual(impedimentosDaNovaTentativa(semNada), []);
  assert.equal(textoDasTentativas(semNada.tentativas_restantes), "Não restam Tentativas de Pagamento para este Pedido.");
  // Aguardando pagamento, o próprio Pedido segura a unidade e o Item lê
  // indisponível (5.8): isso não é impedimento de nada.
  assert.deepEqual(impedimentosDaNovaTentativa({ ...recusado, status: "AGUARDANDO_PAGAMENTO", itens: [item("Relógio", false)] }), []);
});

test("a rota da nova Tentativa é a do próprio Pedido", () => {
  assert.equal(rotaDaNovaTentativa("0190-a/b"), "/api/v1/pedidos/0190-a%2Fb/tentativas");
});

test("a resposta da nova Tentativa: relê no sucesso e nas recusas do Pedido, e erra no resto", () => {
  const resposta = (status, codigo, mensagem) => ({
    resposta: { ok: status < 300, status },
    json: codigo ? { erro: { codigo, mensagem } } : { id: "x", status: "AGUARDANDO_PAGAMENTO" },
  });
  assert.deepEqual(desfechoDaNovaTentativa(resposta(201)), { tipo: "releitura", aviso: null });
  // Sem Estoque e sem Tentativa relêem e avisam: a unidade pode ter voltado
  // antes da releitura, e o clique não pode parecer que não fez nada.
  for (const codigo of ["ESTOQUE_INSUFICIENTE", "TETO_DE_TENTATIVAS"]) {
    assert.deepEqual(desfechoDaNovaTentativa(resposta(409, codigo, "m")), { tipo: "releitura", aviso: "m" }, codigo);
  }
  // A corrida relê calada: o Pedido já anda.
  assert.deepEqual(desfechoDaNovaTentativa(resposta(409, "ESTADO_JA_AVANCADO", "m")), { tipo: "releitura", aviso: null });
  assert.deepEqual(desfechoDaNovaTentativa(resposta(401, "SESSAO_INVALIDA", "m")), { tipo: "semSessao" });
  assert.deepEqual(desfechoDaNovaTentativa(resposta(404, "NAO_ENCONTRADO", "Recurso não encontrado.")), {
    tipo: "erro",
    mensagem: "Recurso não encontrado.",
  });
  // Um 409 que não é do Pedido — ou uma resposta sem envelope — não relê em
  // silêncio: diz que falhou.
  assert.deepEqual(desfechoDaNovaTentativa(resposta(409, "TRANSICAO_INVALIDA", "Esta mudança…")), {
    tipo: "erro",
    mensagem: "Esta mudança…",
  });
  assert.deepEqual(desfechoDaNovaTentativa({ resposta: { ok: false, status: 502 }, json: null }), {
    tipo: "erro",
    mensagem: FALHA_NA_NOVA_TENTATIVA,
  });
});
