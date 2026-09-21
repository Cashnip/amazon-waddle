// A escolha de Endereço do checkout (5.2): as linhas da matriz que são regra.
// O que o Go decide (formato, teto, dono) não tem teste aqui: é dele.
import { test } from "node:test";
import assert from "node:assert/strict";

const {
  CHAVE_DE_IDEMPOTENCIA,
  CHAVE_DO_ENDERECO,
  aberturaDaRevisao,
  aberturaDoPassoEndereco,
  chaveDeIdempotencia,
  desvioDaAberturaDaRevisao,
  desvioDaRevisao,
  enderecoEscolhido,
  ENTRADA_NO_CHECKOUT,
  podeContinuar,
  razoesDoContinuar,
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

// O desvio da Revisão (5.5): leitura pura sobre o Carrinho que ela já pediu.
// Quem reporta e confirma é `POST /api/v1/checkout/entrada`, no passo
// Endereço — aqui não há escrita nenhuma.
const linha = (extras) => ({
  id: "i1",
  produto_id: "p1",
  quantidade: 1,
  visivel: true,
  nome: "Chaleira",
  imagem_url: "",
  preco_centavos: 5000,
  preco_visto_centavos: 5000,
  estoque_disponivel: 10,
  preco_mudou: false,
  bloqueio: "",
  ...extras,
});
const carrinhoCom = (...itens) => ({
  itens,
  unidades: itens.length,
  subtotal_centavos: 0,
  frete_isencao_centavos: 29900,
});

test("Carrinho limpo: a Revisão abre", () => {
  assert.equal(desvioDaRevisao(carrinhoCom(linha({}), linha({ id: "i2" }))), null);
});

test("linha acima do Estoque: volta ao Carrinho, que é onde estão Ajustar e Remover", () => {
  const cheia = linha({ id: "i2", bloqueio: "acima_do_estoque", estoque_disponivel: 0 });
  assert.equal(desvioDaRevisao(carrinhoCom(linha({}), cheia)), "/carrinho");
});

test("Carrinho só de Produtos invisíveis não é vazio, e não chega à Revisão", () => {
  const invisivel = linha({
    visivel: false,
    nome: "",
    preco_centavos: 0,
    estoque_disponivel: 0,
    bloqueio: "indisponivel",
  });
  assert.equal(desvioDaRevisao(carrinhoCom(invisivel)), "/carrinho");
});

test("preço mudado depois da entrada: volta ao passo Endereço, que é quem confirma", () => {
  const mudada = linha({ preco_mudou: true, preco_visto_centavos: 5000, preco_centavos: 6000 });
  assert.equal(desvioDaRevisao(carrinhoCom(mudada)), "/checkout/endereco");
});

// Confirmar preço não desbloqueia: as duas listas são independentes, e quem
// manda é o bloqueio — o passo Endereço não resolveria a linha bloqueada.
test("bloqueio e preço mudado na mesma linha: o bloqueio decide", () => {
  const duas = linha({ preco_mudou: true, bloqueio: "acima_do_estoque" });
  assert.equal(desvioDaRevisao(carrinhoCom(duas)), "/carrinho");
});

// A linha invisível não tem preço a confirmar, e o visto dela nunca é tocado:
// um `preco_mudou` nela não pode mandar o Comprador ao passo Endereço.
test("linha invisível não desvia ao passo Endereço por preço", () => {
  const invisivel = linha({ id: "i2", visivel: false, nome: "", preco_centavos: 0, preco_mudou: true });
  assert.equal(desvioDaRevisao(carrinhoCom(linha({}), invisivel)), null);
});

// A fiação das duas telas do checkout (5.5). Montar componente React pede
// Testing Library e um DOM, que o repositório não tem; o que dá para prender
// aqui — e é o que carrega a estória — são as decisões que os dois `useEffect`
// delegam a funções puras: por qual rota o passo Endereço abre, em que ordem
// as guardas correm, e quando o Continuar libera.

const ok = (json) => ({ resposta: { ok: true, status: 200 }, json });
const naoOk = (status, mensagem) => ({
  resposta: { ok: false, status },
  json: mensagem === undefined ? null : { erro: { mensagem } },
});

// **A rota da abertura é a que escreve.** Trocá-la por `GET /api/v1/carrinho`
// é uma linha, e derruba a estória inteira sem derrubar mais nada.
test("o passo Endereço abre pela rota que escreve, e por POST", () => {
  assert.equal(ENTRADA_NO_CHECKOUT.rota, "/api/v1/checkout/entrada");
  assert.equal(ENTRADA_NO_CHECKOUT.metodo, "POST");
  assert.notEqual(ENTRADA_NO_CHECKOUT.rota, "/api/v1/carrinho");
});

test("abertura do passo Endereço: entrada e lista ok, tudo aplicado", () => {
  const a = aberturaDoPassoEndereco(ok(carrinhoCom(linha({}))), ok([casa, trabalho]));
  assert.equal(a.semSessao, false);
  assert.equal(a.destino, null);
  assert.equal(a.erro, null);
  assert.equal(a.carrinho.itens.length, 1);
  assert.deepEqual(a.enderecos, [casa, trabalho]);
});

test("abertura do passo Endereço: 401 de qualquer uma das duas vai ao Login", () => {
  for (const par of [
    [naoOk(401), ok([casa])],
    [ok(carrinhoCom(linha({}))), naoOk(401)],
  ]) {
    const a = aberturaDoPassoEndereco(...par);
    assert.equal(a.semSessao, true);
    assert.equal(a.carrinho, null);
  }
});

test("abertura do passo Endereço: entrada recusada não inventa relatório", () => {
  const a = aberturaDoPassoEndereco(naoOk(500, "Erro do servidor."), ok([casa]));
  assert.equal(a.carrinho, null);
  assert.equal(a.erro, "Erro do servidor.");
  assert.equal(a.destino, null);
});

test("abertura do passo Endereço: Carrinho vazio volta ao Carrinho", () => {
  const a = aberturaDoPassoEndereco(ok(carrinhoCom()), ok([casa]));
  assert.equal(a.destino, "/carrinho");
});

// **A linha do P1.** A entrada já comitou a ciência quando a lista de
// Endereços falha: perder o relatório aqui é a mudança de preço sumindo sem
// ninguém ver — o AD-17 do lado do cliente. O relatório e o aviso convivem.
test("lista de Endereços falha depois de a entrada comitar: o relatório sobrevive", () => {
  const mudada = linha({ preco_mudou: true, preco_visto_centavos: 5000, preco_centavos: 6000 });
  const a = aberturaDoPassoEndereco(ok(carrinhoCom(mudada)), naoOk(500, "Não foi possível ler os Endereços."));
  assert.notEqual(a.carrinho, null, "o relatório da entrada não pode ser descartado");
  assert.equal(a.carrinho.itens[0].preco_mudou, true);
  assert.equal(a.erro, "Não foi possível ler os Endereços.");
  assert.equal(a.enderecos, null);
});

// E o mesmo no desvio: o Carrinho vazio também não apaga o que já foi gravado.
test("Carrinho vazio: o desvio leva o relatório junto", () => {
  const a = aberturaDoPassoEndereco(ok(carrinhoCom()), ok([casa]));
  assert.notEqual(a.carrinho, null);
  assert.equal(a.carrinho.itens.length, 0);
});

test("sem envelope na resposta, a mensagem é a da tela", () => {
  const a = aberturaDoPassoEndereco(naoOk(500), ok([casa]));
  assert.equal(a.erro, "Não foi possível abrir o Carrinho.");
});

// **A trava do avanço.** `!marcado || !podeAvancar`: trocar por `!marcado`
// sozinho reabre o furo que a 5.5 existe para fechar.
test("o Continuar só libera com Endereço marcado E Carrinho liberado", () => {
  assert.equal(podeContinuar("e1", true), true);
  assert.equal(podeContinuar(null, true), false);
  assert.equal(podeContinuar("e1", false), false);
  assert.equal(podeContinuar(null, false), false);
});

test("as razões do Continuar acumulam, e as duas aparecem quando as duas valem", () => {
  assert.deepEqual(razoesDoContinuar(0, 0), []);
  assert.equal(razoesDoContinuar(1, 0).length, 1);
  assert.equal(razoesDoContinuar(0, 1)[0], "Confirme o novo preço para continuar.");
  assert.equal(razoesDoContinuar(0, 2)[0], "Confirme os novos preços para continuar.");
  assert.equal(razoesDoContinuar(2, 2).length, 2);
});

// **A ordem da abertura da Revisão.** Vazio primeiro, revalidação depois: um
// Carrinho vazio não tem preço a confirmar, e mandá-lo ao passo Endereço seria
// um beco.
test("abertura da Revisão: vazio decide antes da revalidação", () => {
  assert.equal(desvioDaAberturaDaRevisao(carrinhoCom()), "/carrinho");
  const mudada = linha({ preco_mudou: true, preco_visto_centavos: 5000, preco_centavos: 6000 });
  assert.equal(desvioDaAberturaDaRevisao(carrinhoCom(mudada)), "/checkout/endereco");
  assert.equal(desvioDaAberturaDaRevisao(carrinhoCom(linha({}))), null);
});

test("abertura da Revisão: o desvio corre antes da guarda da lista de Endereços", () => {
  const bloqueada = linha({ bloqueio: "indisponivel", visivel: false, nome: "", preco_centavos: 0 });
  const a = aberturaDaRevisao(ok(carrinhoCom(bloqueada)), naoOk(500, "Não foi possível ler os Endereços."));
  assert.equal(a.destino, "/carrinho", "o bloqueio manda de volta mesmo com a lista falhando");
  assert.equal(a.erro, null);
});

test("abertura da Revisão: sem bloqueio e sem preço mudado, ela abre", () => {
  const a = aberturaDaRevisao(ok(carrinhoCom(linha({}))), ok([casa]));
  assert.equal(a.destino, null);
  assert.equal(a.erro, null);
  assert.deepEqual(a.enderecos, [casa]);
});

test("abertura da Revisão: 401 vai ao Login, e o Carrinho recusado vira aviso", () => {
  assert.equal(aberturaDaRevisao(naoOk(401), ok([casa])).semSessao, true);
  assert.equal(aberturaDaRevisao(naoOk(500, "Caiu."), ok([casa])).erro, "Caiu.");
});
