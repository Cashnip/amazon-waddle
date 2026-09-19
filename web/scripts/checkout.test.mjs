// A escolha de Endereço do checkout (5.2): as linhas da matriz que são regra.
// O que o Go decide (formato, teto, dono) não tem teste aqui: é dele.
import { test } from "node:test";
import assert from "node:assert/strict";

const { CHAVE_DO_ENDERECO, enderecoEscolhido, enderecoMarcado, guardarEscolha, lerEscolha } = await import(
  new URL("../lib/checkout.ts", import.meta.url).href
);

const casa = { id: "e1", cidade: "São Paulo" };
const trabalho = { id: "e2", cidade: "Campinas" };

// Um `sessionStorage` de mentira, em memória.
function memoria() {
  const dados = new Map();
  const armazenamento = {
    getItem: (k) => (dados.has(k) ? dados.get(k) : null),
    setItem: (k, v) => void dados.set(k, String(v)),
  };
  return { dados, obter: () => armazenamento };
}

// O armazenamento que lança: janela privada, site bloqueado, cota cheia.
const lanca = () => {
  throw new Error("SecurityError");
};
const lancaNoUso = () => ({
  getItem: () => {
    throw new Error("SecurityError");
  },
  setItem: () => {
    throw new Error("QuotaExceededError");
  },
});

test("com Endereços e nada guardado: o primeiro da lista vem marcado", () => {
  assert.equal(enderecoMarcado([casa, trabalho], null), "e1");
});

test("escolha guardada presente na lista: ela vem marcada", () => {
  assert.equal(enderecoMarcado([casa, trabalho], "e2"), "e2");
});

test("guardado sumiu (removido em Meus endereços): o primeiro vem marcado", () => {
  assert.equal(enderecoMarcado([casa, trabalho], "removido"), "e1");
});

test("lista vazia: nada marcado, guardado ou não", () => {
  assert.equal(enderecoMarcado([], null), null);
  assert.equal(enderecoMarcado([], "e1"), null);
});

test("o Endereço recém-cadastrado vem marcado depois de recarregar a lista", () => {
  const novo = { id: "e3", cidade: "Santos" };
  assert.equal(enderecoMarcado([casa, trabalho, novo], "e3"), "e3");
});

test("recém-cadastrado vence o guardado", () => {
  const novo = { id: "e3", cidade: "Santos" };
  assert.equal(enderecoMarcado([casa, trabalho, novo], "e1", "e3"), "e3");
});

test("preferido fora da lista: cai no guardado, e sem ele no primeiro", () => {
  assert.equal(enderecoMarcado([casa, trabalho], "e2", "sumiu"), "e2");
  assert.equal(enderecoMarcado([casa, trabalho], null, "sumiu"), "e1");
  assert.equal(enderecoMarcado([], "e2", "e3"), null);
});

test("guardar e ler: ida e volta, só o id, na chave do checkout", () => {
  const { dados, obter } = memoria();
  assert.equal(lerEscolha(obter), null);
  assert.equal(guardarEscolha("e2", obter), true);
  assert.equal(lerEscolha(obter), "e2");
  // Só o id: nenhum Frete, CEP ou total atravessa a ida ao Carrinho.
  assert.deepEqual([...dados.entries()], [[CHAVE_DO_ENDERECO, "e2"]]);
});

test("sessionStorage que lança equivale a nada guardado", () => {
  for (const obter of [lanca, lancaNoUso]) {
    assert.equal(lerEscolha(obter), null);
    assert.equal(guardarEscolha("e1", obter), false);
  }
  // E a regra segue sem ele: o primeiro da lista.
  assert.equal(enderecoMarcado([casa, trabalho], lerEscolha(lanca)), "e1");
});

test("valor vazio guardado equivale a nada guardado", () => {
  const { obter } = memoria();
  guardarEscolha("", obter);
  assert.equal(lerEscolha(obter), null);
});

test("sem navegador, o padrão também não estoura", () => {
  // No Node não há `window`: o acesso lança dentro do `try`.
  assert.equal(lerEscolha(), null);
  assert.equal(guardarEscolha("e1"), false);
});

test("Revisão: só o guardado, e só se está na lista — sem cair no primeiro", () => {
  assert.equal(enderecoEscolhido([casa, trabalho], "e2"), trabalho);
  assert.equal(enderecoEscolhido([casa, trabalho], null), null);
  assert.equal(enderecoEscolhido([casa, trabalho], "removido"), null);
  assert.equal(enderecoEscolhido([], "e1"), null);
});
