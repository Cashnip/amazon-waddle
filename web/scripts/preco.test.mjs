// A conversão do preço é o único caminho monetário do navegador: nenhum ponto
// flutuante entre o texto e os centavos (AD-3).
import { test } from "node:test";
import assert from "node:assert/strict";
import { reaisParaCentavos, centavosParaReais, formatarPreco } from "../lib/preco.ts";

test("reais em texto viram centavos inteiros", () => {
  const casos = {
    "12,90": 1290,
    "12,9": 1290,
    "12": 1200,
    "0,07": 7,
    "1.299,90": 129990,
    " R$ 1.000.000,00 ": 100000000,
    "4,35": 435, // 4.35 * 100 em ponto flutuante dá 434.99999999999994
  };
  for (const [texto, centavos] of Object.entries(casos)) {
    assert.equal(reaisParaCentavos(texto), centavos, texto);
  }
});

test("texto fora do formato brasileiro é recusado", () => {
  for (const texto of ["", "abc", "12.90", "12,905", "1.29,00", "-5", "1,2,3"]) {
    assert.equal(reaisParaCentavos(texto), null, texto);
  }
});

test("centavos voltam ao texto do formulário e da tela", () => {
  assert.equal(centavosParaReais(1290), "12,90");
  assert.equal(centavosParaReais(7), "0,07");
  assert.equal(formatarPreco(129990), "R$ 1.299,90");
});
