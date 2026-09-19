// A linha "Volta do Login" da matriz da 3.6: o `?quantidade` limitado.
import { test } from "node:test";
import assert from "node:assert/strict";

const { quantidadeDaUrl, quantidadeMaxima } = await import(
  new URL("../lib/quantidade.ts", import.meta.url).href
);

test("quantidade: dentro do intervalo passa, acima vira o teto", () => {
  assert.equal(quantidadeDaUrl("3", 5), 3);
  assert.equal(quantidadeDaUrl("50", 5), 5);
  assert.equal(quantidadeDaUrl("50", 40), 10);
  assert.equal(quantidadeDaUrl(["4", "9"], 5), 4);
});

test("quantidade: inválida vira 1", () => {
  for (const v of ["abc", "0", "-2", "1.5", "", undefined, "99999999999999999999"]) {
    assert.ok(quantidadeDaUrl(v, 5) >= 1 && quantidadeDaUrl(v, 5) <= 5, String(v));
  }
  for (const v of ["abc", "0", "-2", "1.5", "", undefined]) assert.equal(quantidadeDaUrl(v, 5), 1, String(v));
});

test("quantidade: o máximo é min(10, disponível), nunca abaixo de 1", () => {
  assert.equal(quantidadeMaxima(3), 3);
  assert.equal(quantidadeMaxima(25), 10);
  assert.equal(quantidadeMaxima(0), 1);
});
