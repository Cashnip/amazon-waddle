// A aritmética da edição otimista da 4.3 e a leitura do campo de quantidade. O
// que o Go decide (teto, Estoque, preço) não tem teste aqui: é dele.
import { test } from "node:test";
import assert from "node:assert/strict";

const { comQuantidade, parcelaDe, quantidadeDoCampo, subtotalDe, unidadesDe } = await import(
  new URL("../lib/carrinho.ts", import.meta.url).href
);

const chaleira = { id: "a", produto_id: "p1", quantidade: 3, visivel: true, nome: "Chaleira", imagem_url: "", preco_centavos: 5990 };
const bule = { id: "b", produto_id: "p2", quantidade: 2, visivel: true, nome: "Bule", imagem_url: "", preco_centavos: 3005 };
const fora = { id: "c", produto_id: "p3", quantidade: 4, visivel: false, nome: "", imagem_url: "", preco_centavos: 0 };

test("subtotal: soma de parcelas em centavos inteiros, sem a linha indisponível", () => {
  assert.equal(parcelaDe(chaleira), 17970);
  assert.equal(parcelaDe(fora), 0);
  assert.equal(subtotalDe([chaleira, bule, fora]), 17970 + 6010);
});

test("unidades: somam todas as linhas, inclusive a indisponível", () => {
  assert.equal(unidadesDe([chaleira, bule, fora]), 9);
});

test("edição otimista: a quantidade nova refaz unidades e subtotal, e o resto fica", () => {
  const antes = { itens: [chaleira, bule, fora], unidades: 9, subtotal_centavos: 23980 };
  const depois = comQuantidade(antes, "a", 1);
  assert.equal(depois.itens[0].quantidade, 1);
  assert.equal(depois.itens[1], bule);
  assert.equal(depois.unidades, 7);
  assert.equal(depois.subtotal_centavos, 5990 + 6010);
  // O estado anterior fica intacto: é ele que a tela repõe quando o servidor recusa.
  assert.equal(antes.itens[0].quantidade, 3);
  assert.equal(antes.subtotal_centavos, 23980);
});

test("campo de quantidade: inteiro sem sinal passa, o resto vira null", () => {
  assert.equal(quantidadeDoCampo("3"), 3);
  assert.equal(quantidadeDoCampo(" 07 "), 7);
  assert.equal(quantidadeDoCampo("0"), 0);
  for (const v of ["", "abc", "-1", "1.5", "1,5", "+2", "99999999999999999999"]) {
    assert.equal(quantidadeDoCampo(v), null, JSON.stringify(v));
  }
});
