// O formulário de Endereço compartilhado (5.2): a escolha de método e rota, a
// leitura do erro do envelope e a máscara do CEP. O que o Go decide (formato,
// UF, teto) não tem teste aqui: é dele.
import { test } from "node:test";
import assert from "node:assert/strict";

const { CAMPOS, FALHA_AO_SALVAR, comMascara, erroDeSalvar, requisicaoDeSalvar } = await import(
  new URL("../lib/endereco.ts", import.meta.url).href
);

const casa = {
  id: "e1", destinatario: "Ana", cep: "01310100", logradouro: "Av. Paulista", numero: "1000",
  complemento: "", bairro: "Bela Vista", cidade: "São Paulo", uf: "SP",
};

test("salvar: sem Endereço é cadastro, POST na coleção", () => {
  assert.deepEqual(requisicaoDeSalvar(null), { url: "/api/v1/enderecos", method: "POST" });
});

test("salvar: com Endereço é edição, PUT no id codificado", () => {
  assert.deepEqual(requisicaoDeSalvar(casa), { url: "/api/v1/enderecos/e1", method: "PUT" });
  assert.deepEqual(requisicaoDeSalvar({ ...casa, id: "a/b c" }), {
    url: "/api/v1/enderecos/a%2Fb%20c",
    method: "PUT",
  });
});

test("erro: o campo do envelope e a mensagem do servidor", () => {
  const corpo = { erro: { codigo: "VALIDACAO", mensagem: "CEP inválido.", dados: { campo: "cep" } } };
  assert.deepEqual(erroDeSalvar(corpo), { campo: "cep", mensagem: "CEP inválido." });
  for (const campo of CAMPOS) {
    assert.equal(erroDeSalvar({ erro: { mensagem: "x", dados: { campo } } }).campo, campo);
  }
});

test("erro: campo desconhecido ou ausente cai no alerta, com a mensagem do servidor", () => {
  assert.deepEqual(erroDeSalvar({ erro: { mensagem: "Limite de Endereços.", dados: { campo: "senha" } } }), {
    campo: null,
    mensagem: "Limite de Endereços.",
  });
  // O 409 do teto não nomeia campo nenhum.
  assert.deepEqual(erroDeSalvar({ erro: { mensagem: "Limite de Endereços." } }), {
    campo: null,
    mensagem: "Limite de Endereços.",
  });
});

test("erro: corpo sem envelope usa a mensagem padrão", () => {
  assert.equal(FALHA_AO_SALVAR, "Não foi possível salvar o Endereço.");
  for (const corpo of [null, undefined, {}, { erro: {} }, { erro: { mensagem: "" } }, "html"]) {
    assert.deepEqual(erroDeSalvar(corpo), { campo: null, mensagem: FALHA_AO_SALVAR }, JSON.stringify(corpo));
  }
});

test("CEP: oito dígitos ganham o hífen; o resto passa como veio", () => {
  assert.equal(comMascara("01310100"), "01310-100");
  assert.equal(comMascara("01310-100"), "01310-100");
  assert.equal(comMascara("0131"), "0131");
  assert.equal(comMascara(""), "");
});
