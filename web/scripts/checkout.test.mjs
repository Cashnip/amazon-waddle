// A escolha de Endereço do checkout (5.2): as linhas da matriz que são regra.
// O que o Go decide (formato, teto, dono) não tem teste aqui: é dele.
import { test } from "node:test";
import assert from "node:assert/strict";

const {
  CHAVE_DE_IDEMPOTENCIA,
  CHAVE_DO_ENDERECO,
  chaveDeIdempotencia,
  enderecoEscolhido,
  enderecoMarcado,
  freteGratis,
  guardarEscolha,
  lerEscolha,
  limparCheckout,
  pareceUUIDV4,
  rotaDoFrete,
  uuidNovo,
} = await import(
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
    removeItem: (k) => void dados.delete(k),
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
  removeItem: () => {
    throw new Error("SecurityError");
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

// O Frete (5.3): a tela só monta a rota e escolhe a palavra. Região, valor e
// isenção são do Go, e estão testados lá.
test("a rota do Frete leva o Endereço escolhido, escapado", () => {
  assert.equal(rotaDoFrete("e1"), "/api/v1/frete?endereco_id=e1");
  assert.equal(rotaDoFrete("a b&c"), "/api/v1/frete?endereco_id=a%20b%26c");
});

test("Frete zero é Grátis; qualquer outro valor não", () => {
  assert.equal(freteGratis({ frete_centavos: 0 }), true);
  assert.equal(freteGratis({ frete_centavos: 1500 }), false);
});

// A chave de idempotência (5.4). Ela é estado de tela — uma por tentativa de
// checkout —, e o que se prova aqui é o que a matriz chama de regra: nasce na
// primeira entrada, sobrevive ao recarregamento, e sem armazenamento não
// existe. Quem a envia, e quem chama `limparCheckout`, é a 5.6.
const UUIDV4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

// O `crypto` de contexto não seguro — um IP da LAN na apresentação —, que não
// expõe `randomUUID`. Bytes todos 0xff para a forma do UUID ficar previsível.
const semRandomUUID = () => ({
  getRandomValues: (a) => {
    a.fill(0xff);
    return a;
  },
});

test("primeira entrada na Revisão: gera a chave e a guarda", () => {
  const { dados, obter } = memoria();
  const chave = chaveDeIdempotencia(obter);
  // Não-nulo antes da forma: `assert.match` sobre `null` falha com erro de
  // tipo, e não com a mensagem que este teste existe para dar.
  assert.notEqual(chave, null);
  assert.match(chave, UUIDV4);
  assert.deepEqual([...dados.entries()], [[CHAVE_DE_IDEMPOTENCIA, chave]]);
});

test("recarregar a Revisão: a mesma chave continua, nenhuma nova é gerada", () => {
  const { dados, obter } = memoria();
  const primeira = chaveDeIdempotencia(obter);
  assert.notEqual(primeira, null);
  // Recarregar, voltar ao Carrinho e retornar e trocar o Endereço passam todos
  // pela entrada da Revisão, que é esta chamada: o que se prova aqui é que
  // entrar de novo, quantas vezes for, não troca a chave. As navegações em si
  // são teste de tela, que o `web/` ainda não tem bancada para montar.
  for (let i = 0; i < 3; i++) assert.equal(chaveDeIdempotencia(obter), primeira);
  assert.equal(dados.size, 1);
});

test("sessionStorage indisponível: sem chave, e a tela avisa em vez de prometer", () => {
  for (const obter of [lanca, lancaNoUso]) {
    assert.equal(chaveDeIdempotencia(obter), null);
  }
  // Sem navegador o padrão também não estoura.
  assert.equal(chaveDeIdempotencia(), null);
});

test("limparCheckout apaga as duas chaves, e só elas", () => {
  const { dados, obter } = memoria();
  guardarEscolha("e2", obter);
  const chave = chaveDeIdempotencia(obter);
  dados.set("azamon:outra", "fica");
  assert.equal(dados.get(CHAVE_DO_ENDERECO), "e2");
  assert.equal(dados.get(CHAVE_DE_IDEMPOTENCIA), chave);

  limparCheckout(obter);
  assert.deepEqual([...dados.entries()], [["azamon:outra", "fica"]]);
  // Apagada, a entrada seguinte é uma tentativa nova: chave nova.
  assert.notEqual(chaveDeIdempotencia(obter), chave);
});

test("limparCheckout sobre armazenamento que lança não estoura", () => {
  limparCheckout(lanca);
  limparCheckout(lancaNoUso);
  limparCheckout();
});

// Um armazenamento em memória que recusa apagar UMA das chaves. É o que prova
// que cada chave tem o seu `try`: com um `try` só em volta das duas, a que vem
// depois da que lançou ficaria para trás — e a chave de idempotência
// sobrevivente faria o próximo checkout reusar a chave do Pedido anterior.
function memoriaQueRecusaApagar(chaveRuim) {
  const { dados, obter } = memoria();
  const bom = obter();
  const armazenamento = {
    ...bom,
    removeItem: (k) => {
      if (k === chaveRuim) throw new Error("SecurityError");
      bom.removeItem(k);
    },
  };
  return { dados, obter: () => armazenamento };
}

test("limparCheckout: a chave que lança não leva a outra junto, nos dois sentidos", () => {
  for (const chaveRuim of [CHAVE_DO_ENDERECO, CHAVE_DE_IDEMPOTENCIA]) {
    const { dados, obter } = memoriaQueRecusaApagar(chaveRuim);
    guardarEscolha("e2", obter);
    chaveDeIdempotencia(obter);
    assert.equal(dados.size, 2);

    limparCheckout(obter);
    // Sobra exatamente a que recusou ser apagada, e nenhuma outra.
    assert.deepEqual([...dados.keys()], [chaveRuim]);
  }
});

test("a forma da chave é um UUIDv4, e duas seguidas diferem", () => {
  const a = uuidNovo();
  assert.match(a, UUIDV4);
  assert.notEqual(a, uuidNovo());
});

test("sem randomUUID (contexto não seguro), a queda monta o UUIDv4 à mão", () => {
  // Versão no nibble alto do sétimo byte e variante nos dois bits altos do
  // nono: com 16 bytes 0xff, só esses dois nibbles não saem "f".
  assert.equal(uuidNovo(semRandomUUID), "ffffffff-ffff-4fff-bfff-ffffffffffff");
  assert.match(uuidNovo(semRandomUUID), UUIDV4);
});

test("a chave guardada sai da queda quando não há randomUUID", () => {
  const { dados, obter } = memoria();
  assert.equal(chaveDeIdempotencia(obter, semRandomUUID), "ffffffff-ffff-4fff-bfff-ffffffffffff");
  assert.equal(dados.get(CHAVE_DE_IDEMPOTENCIA), "ffffffff-ffff-4fff-bfff-ffffffffffff");
});

// O `sessionStorage` é do navegador: outra aba, uma extensão ou um dedo no
// console escrevem nele. Na 5.6 o valor vira o cabeçalho `Idempotency-Key`, e
// um caractere fora do *token* HTTP — ou um valor quilométrico — quebra a
// requisição em vez de falhar com jeito. Fora da forma, vale como nada
// guardado.
test("chave guardada fora da forma de UUIDv4 é descartada e regerada", () => {
  const lixo = [
    "",
    "   ",
    "nao-e-uuid",
    "x".repeat(5000),
    "chave com espaço",
    "linha\nquebrada",
    // Versão 1, variante fora de 8–b, maiúscula e com chaves em volta: todas
    // são UUID plausíveis que não são o que geramos.
    "3f2504e0-4f89-11d3-9a0c-0305e82c3301",
    "ffffffff-ffff-4fff-cfff-ffffffffffff",
    "FFFFFFFF-FFFF-4FFF-BFFF-FFFFFFFFFFFF",
    "{ffffffff-ffff-4fff-bfff-ffffffffffff}",
    "ffffffff-ffff-4fff-bfff-ffffffffffff ",
  ];
  for (const valor of lixo) {
    const { dados, obter } = memoria();
    dados.set(CHAVE_DE_IDEMPOTENCIA, valor);
    const chave = chaveDeIdempotencia(obter);
    assert.notEqual(chave, null);
    assert.match(chave, UUIDV4);
    assert.notEqual(chave, valor);
    // E o lixo foi sobrescrito, não deixado ao lado.
    assert.equal(dados.get(CHAVE_DE_IDEMPOTENCIA), chave);
  }
});

test("pareceUUIDV4 aceita o que uuidNovo gera, pelos dois caminhos", () => {
  assert.equal(pareceUUIDV4(uuidNovo()), true);
  assert.equal(pareceUUIDV4(uuidNovo(semRandomUUID)), true);
  assert.equal(pareceUUIDV4("nao-e-uuid"), false);
});
