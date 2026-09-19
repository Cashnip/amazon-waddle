// As linhas de URL da matriz da 3.10: a busca global e a Faixa de Categorias.
import { test } from "node:test";
import assert from "node:assert/strict";

const { destinoDaBusca, destinoDaCategoria } = await import(new URL("../lib/busca.ts", import.meta.url).href);

const ID = "0b6f1c1e-2a3b-4c5d-8e9f-001122334455";

test("busca: termo sozinho, com Categoria e vazio", () => {
  assert.equal(destinoDaBusca("cafe", ""), "/?termo=cafe");
  assert.equal(destinoDaBusca("  cafe ", ""), "/?termo=cafe");
  assert.equal(destinoDaBusca("cafe", ID), `/?termo=cafe&categoria=${ID}`);
  assert.equal(destinoDaBusca("", ""), "/");
  assert.equal(destinoDaBusca("   ", ID), `/?categoria=${ID}`);
});

test("busca: termo com caracteres especiais é codificado", () => {
  assert.equal(destinoDaBusca("café & 50%", ""), "/?termo=caf%C3%A9+%26+50%25");
});

test("faixa: o link da Categoria", () => {
  assert.equal(destinoDaCategoria(ID), `/?categoria=${ID}`);
});
