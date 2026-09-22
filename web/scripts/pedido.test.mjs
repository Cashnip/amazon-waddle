// A tela do Pedido (5.8): as linhas da matriz que são leitura da resposta do
// Go. O que o Go decide (prazo, terminal, tentativas, disponível) é testado
// lá, em api/pedido_detalhe_test.go.
import { test } from "node:test";
import assert from "node:assert/strict";

const {
  INTERVALO_AGUARDANDO_MS,
  INTERVALO_AVANCANDO_MS,
  intervaloDaConsulta,
  motivoDaRecusa,
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
