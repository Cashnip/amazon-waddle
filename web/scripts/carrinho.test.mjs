// A aritmética da edição otimista da 4.3 e a leitura do campo de quantidade. O
// que o Go decide (teto, Estoque, preço) não tem teste aqui: é dele.
import { test } from "node:test";
import assert from "node:assert/strict";

const {
  comPrecosConfirmados,
  comQuantidade,
  parcelaDe,
  quantidadeDoCampo,
  revalidacaoDe,
  subtotalDe,
  textoDoBloqueio,
  unidadesDe,
  unidadesTexto,
} = await import(
  new URL("../lib/carrinho.ts", import.meta.url).href
);

const chaleira = {
  id: "a", produto_id: "p1", quantidade: 3, visivel: true, nome: "Chaleira", imagem_url: "", preco_centavos: 5990,
  preco_visto_centavos: 5990, estoque_disponivel: 10, preco_mudou: false, bloqueio: "",
};
const bule = {
  id: "b", produto_id: "p2", quantidade: 2, visivel: true, nome: "Bule", imagem_url: "", preco_centavos: 3005,
  preco_visto_centavos: 3005, estoque_disponivel: 10, preco_mudou: false, bloqueio: "",
};
const fora = {
  id: "c", produto_id: "p3", quantidade: 4, visivel: false, nome: "", imagem_url: "", preco_centavos: 0,
  preco_visto_centavos: 0, estoque_disponivel: 0, preco_mudou: false, bloqueio: "indisponivel",
};

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

const carrinhoDe = (...itens) => ({ itens, unidades: unidadesDe(itens), subtotal_centavos: subtotalDe(itens) });
const maisCara = { ...chaleira, preco_centavos: 6990, preco_mudou: true };
const escassa = { ...bule, quantidade: 4, estoque_disponivel: 2, bloqueio: "acima_do_estoque" };
const esgotada = { ...bule, id: "d", estoque_disponivel: 0, bloqueio: "indisponivel" };

test("revalidação: Carrinho limpo pode avançar", () => {
  const r = revalidacaoDe(carrinhoDe(chaleira, bule), {});
  assert.deepEqual(r.bloqueios, []);
  assert.deepEqual(r.mudancas, []);
  assert.equal(r.podeAvancar, true);
});

test("revalidação: preço mudado pede confirmação, e confirmar o libera", () => {
  const carrinho = carrinhoDe(maisCara, bule);
  const antes = revalidacaoDe(carrinho, {});
  assert.deepEqual(antes.mudancas, [maisCara]);
  assert.equal(antes.podeAvancar, false);

  const depois = revalidacaoDe(carrinho, comPrecosConfirmados({}, antes.mudancas));
  assert.deepEqual(depois.mudancas, []);
  assert.equal(depois.podeAvancar, true);
});

test("revalidação: se o preço muda de novo, o aviso volta", () => {
  const confirmados = comPrecosConfirmados({}, [maisCara]);
  const outraVez = { ...maisCara, preco_centavos: 7500 };
  assert.deepEqual(revalidacaoDe(carrinhoDe(outraVez), confirmados).mudancas, [outraVez]);
});

test("revalidação: confirmar o preço não desbloqueia", () => {
  const carrinho = carrinhoDe({ ...escassa, preco_mudou: true, preco_centavos: 3500 }, fora);
  const confirmados = comPrecosConfirmados({}, revalidacaoDe(carrinho, {}).mudancas);
  const r = revalidacaoDe(carrinho, confirmados);
  assert.deepEqual(r.mudancas, []);
  assert.equal(r.bloqueios.length, 2);
  assert.equal(r.podeAvancar, false);
});

test("revalidação: o Produto invisível não tem preço a confirmar", () => {
  const r = revalidacaoDe(carrinhoDe({ ...fora, preco_mudou: true }), {});
  assert.deepEqual(r.mudancas, []);
  assert.equal(r.bloqueios.length, 1);
});

test("texto do bloqueio: nomeia o Produto, diz o disponível e não fala em Reserva", () => {
  assert.equal(textoDoBloqueio(escassa), "Restam 2 unidades de Bule; o Carrinho pede 4.");
  assert.equal(textoDoBloqueio({ ...escassa, estoque_disponivel: 1 }), "Resta 1 unidade de Bule; o Carrinho pede 4.");
  assert.equal(textoDoBloqueio(esgotada), "Bule está indisponível.");
  assert.equal(textoDoBloqueio(fora), "Produto indisponível.");
  assert.equal(unidadesTexto(1), "1 unidade");
  assert.equal(unidadesTexto(3), "3 unidades");
  for (const l of [escassa, esgotada, fora]) assert.doesNotMatch(textoDoBloqueio(l), /reserv/i);
});
