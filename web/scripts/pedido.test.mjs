// A tela do Pedido (5.8): as linhas da matriz que são leitura da resposta do
// Go. O que o Go decide (prazo, terminal, tentativas, disponível) é testado
// lá, em api/pedido_detalhe_test.go.
import { test } from "node:test";
import assert from "node:assert/strict";

const {
  CORRIDA_DO_CANCELAMENTO,
  FALHA_NA_NOVA_TENTATIVA,
  FALHA_NO_CANCELAMENTO,
  INTERVALO_AGUARDANDO_MS,
  INTERVALO_AVANCANDO_MS,
  PRAZO_DO_CANCELAMENTO_MS,
  desfechoDaNovaTentativa,
  desfechoDoCancelamento,
  fraseDoMotivo,
  fraseSemCancelamento,
  impedimentosDaNovaTentativa,
  intervaloDaConsulta,
  linhaDoTempo,
  motivoDaRecusa,
  podeCancelar,
  podeTentarDeNovo,
  porQueNaoCancela,
  rotaDaNovaTentativa,
  rotaDoCancelamento,
  rotuloDoStatus,
  superficieDoPedido,
  tempoRestante,
  textoDasTentativas,
  textosDoCancelamento,
  ANTIGOS,
  FALHA_NA_TRANSICAO,
  ORDENACOES,
  RECENTES,
  STATUS,
  consultaDaTabela,
  desfechoDaTransicao,
  enderecoDaTabela,
  filtroDaURL,
  intervaloDaTabela,
  rotaDaTabela,
  rotaDaTransicao,
  rotuloDaAcao,
  anuncioDaTabela,
  statusPorPedido,
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

test("a recusa do Provedor tem frase própria, e o motivo desconhecido cai na genérica", () => {
  const nascimento = { de: null, para: "AGUARDANDO_PAGAMENTO", ator: "COMPRADOR", motivo: null, em: "t0" };
  const recusou = {
    de: "AGUARDANDO_PAGAMENTO",
    para: "PAGAMENTO_RECUSADO",
    ator: "PROVEDOR",
    motivo: "RECUSADO_PELO_PROVEDOR",
    em: "t1",
  };
  assert.equal(motivoDaRecusa([nascimento, recusou]), "O Provedor de Pagamento recusou a Tentativa de Pagamento.");
  assert.equal(motivoDaRecusa([nascimento, { ...recusou, motivo: "OUTRO" }]), "O pagamento foi recusado.");
  // Chave herdada de Object.prototype não é motivo conhecido.
  assert.equal(motivoDaRecusa([nascimento, { ...recusou, motivo: "constructor" }]), "O pagamento foi recusado.");
});

// A tripla do AD-18: Status, Tentativas restantes e disponível por Item.
const item = (nome, disponivel) => ({
  produto_id: nome,
  nome,
  quantidade: 1,
  preco_praticado_centavos: 4990,
  disponivel,
});
const recusado = { status: "PAGAMENTO_RECUSADO", tentativas_restantes: 2, itens: [item("Relógio", true), item("Caneca", true)] };

test("tentar de novo só com recusado, Tentativa restante e todo Item disponível", () => {
  assert.equal(podeTentarDeNovo(recusado), true);
  assert.equal(podeTentarDeNovo({ ...recusado, tentativas_restantes: 0 }), false);
  assert.equal(podeTentarDeNovo({ ...recusado, itens: [item("Relógio", true), item("Caneca", false)] }), false);
  for (const status of ["AGUARDANDO_PAGAMENTO", "PAGO", "CANCELADO"]) {
    assert.equal(podeTentarDeNovo({ ...recusado, status }), false, status);
  }
});

test("sem Estoque, a tela nomeia cada Produto; sem Tentativa, quem fala é o texto das Tentativas", () => {
  assert.deepEqual(impedimentosDaNovaTentativa(recusado), []);
  assert.deepEqual(impedimentosDaNovaTentativa({ ...recusado, itens: [item("Relógio", false), item("Caneca", true)] }), [
    { chave: "Relógio", texto: "Não há Estoque disponível de Relógio para uma nova Tentativa de Pagamento." },
  ]);
  // Dois Itens com o mesmo nome congelado: a chave é o Produto, e não a frase.
  const gemeos = [
    { ...item("Caneca", false), produto_id: "p1" },
    { ...item("Caneca", false), produto_id: "p2" },
  ];
  assert.deepEqual(
    impedimentosDaNovaTentativa({ ...recusado, itens: gemeos }).map((i) => i.chave),
    ["p1", "p2"],
  );
  const semNada = { ...recusado, tentativas_restantes: 0, itens: [item("Relógio", false)] };
  assert.deepEqual(impedimentosDaNovaTentativa(semNada), []);
  assert.equal(textoDasTentativas(semNada.tentativas_restantes), "Não restam Tentativas de Pagamento para este Pedido.");
  // Aguardando pagamento, o próprio Pedido segura a unidade e o Item lê
  // indisponível (5.8): isso não é impedimento de nada.
  assert.deepEqual(impedimentosDaNovaTentativa({ ...recusado, status: "AGUARDANDO_PAGAMENTO", itens: [item("Relógio", false)] }), []);
});

test("a rota da nova Tentativa é a do próprio Pedido", () => {
  assert.equal(rotaDaNovaTentativa("0190-a/b"), "/api/v1/pedidos/0190-a%2Fb/tentativas");
});

test("a resposta da nova Tentativa: relê no sucesso e nas recusas do Pedido, e erra no resto", () => {
  const resposta = (status, codigo, mensagem) => ({
    resposta: { ok: status < 300, status },
    json: codigo ? { erro: { codigo, mensagem } } : { id: "x", status: "AGUARDANDO_PAGAMENTO" },
  });
  assert.deepEqual(desfechoDaNovaTentativa(resposta(201)), { tipo: "releitura", aviso: null });
  // Sem Estoque e sem Tentativa relêem e avisam: a unidade pode ter voltado
  // antes da releitura, e o clique não pode parecer que não fez nada.
  for (const codigo of ["ESTOQUE_INSUFICIENTE", "TETO_DE_TENTATIVAS"]) {
    assert.deepEqual(desfechoDaNovaTentativa(resposta(409, codigo, "m")), { tipo: "releitura", aviso: "m" }, codigo);
  }
  // A corrida relê calada: o Pedido já anda.
  assert.deepEqual(desfechoDaNovaTentativa(resposta(409, "ESTADO_JA_AVANCADO", "m")), { tipo: "releitura", aviso: null });
  assert.deepEqual(desfechoDaNovaTentativa(resposta(401, "SESSAO_INVALIDA", "m")), { tipo: "semSessao" });
  assert.deepEqual(desfechoDaNovaTentativa(resposta(404, "NAO_ENCONTRADO", "Recurso não encontrado.")), {
    tipo: "erro",
    mensagem: "Recurso não encontrado.",
  });
  // Um 409 que não é do Pedido — ou uma resposta sem envelope — não relê em
  // silêncio: diz que falhou.
  assert.deepEqual(desfechoDaNovaTentativa(resposta(409, "TRANSICAO_INVALIDA", "Esta mudança…")), {
    tipo: "erro",
    mensagem: "Esta mudança…",
  });
  assert.deepEqual(desfechoDaNovaTentativa({ resposta: { ok: false, status: 502 }, json: null }), {
    tipo: "erro",
    mensagem: FALHA_NA_NOVA_TENTATIVA,
  });
});

test("os sete Status têm rótulo; o que a tela não conhece sai cru", () => {
  assert.equal(rotuloDoStatus("AGUARDANDO_PAGAMENTO"), "Aguardando pagamento");
  assert.equal(rotuloDoStatus("PAGAMENTO_RECUSADO"), "Pagamento recusado");
  assert.equal(rotuloDoStatus("PAGO"), "Pago");
  assert.equal(rotuloDoStatus("SEPARANDO"), "Separando");
  assert.equal(rotuloDoStatus("ENVIADO"), "Enviado");
  assert.equal(rotuloDoStatus("ENTREGUE"), "Entregue");
  assert.equal(rotuloDoStatus("CANCELADO"), "Cancelado");
  // Status que o banco ganhe amanhã aparece feio, e não invisível — e o
  // herdado de Object.prototype não vira rótulo.
  assert.equal(rotuloDoStatus("DEVOLVIDO"), "DEVOLVIDO");
  assert.equal(rotuloDoStatus("constructor"), "constructor");
});

test("a linha do tempo é o histórico na ordem do Go, com rótulo, antes e motivo", () => {
  const historico = [
    { de: null, para: "AGUARDANDO_PAGAMENTO", ator: "COMPRADOR", motivo: null, em: "2026-09-23T10:00:00Z" },
    { de: "AGUARDANDO_PAGAMENTO", para: "PAGAMENTO_RECUSADO", ator: "VARREDURA", motivo: "TEMPO_ESGOTADO", em: "2026-09-23T10:15:00Z" },
    { de: "PAGAMENTO_RECUSADO", para: "AGUARDANDO_PAGAMENTO", ator: "COMPRADOR", motivo: null, em: "2026-09-23T10:16:00Z" },
    { de: "AGUARDANDO_PAGAMENTO", para: "PAGO", ator: "PROVEDOR", motivo: null, em: "2026-09-23T10:17:00Z" },
  ];
  assert.deepEqual(linhaDoTempo(historico), [
    // Nascimento: só o estado novo, sem "antes".
    { status: "Aguardando pagamento", antes: null, motivo: null, em: "2026-09-23T10:00:00Z" },
    { status: "Pagamento recusado", antes: "Aguardando pagamento", motivo: "Tempo de pagamento expirado.", em: "2026-09-23T10:15:00Z" },
    // A nova Tentativa é mais uma linha, e o anterior é o que a distingue.
    { status: "Aguardando pagamento", antes: "Pagamento recusado", motivo: null, em: "2026-09-23T10:16:00Z" },
    { status: "Pago", antes: "Aguardando pagamento", motivo: null, em: "2026-09-23T10:17:00Z" },
  ]);
  assert.deepEqual(linhaDoTempo([]), []);
});

test("motivo desconhecido sai sem frase, e Status desconhecido sai cru", () => {
  const linha = { de: "PAGO", para: "NOVO", ator: "ADMINISTRADOR", motivo: "OUTRO", em: "t" };
  assert.deepEqual(linhaDoTempo([linha]), [{ status: "NOVO", antes: "Pago", motivo: null, em: "t" }]);
  assert.equal(linhaDoTempo([{ ...linha, motivo: "constructor" }])[0].motivo, null);
  assert.equal(fraseDoMotivo(null), null);
  assert.equal(fraseDoMotivo("RECUSADO_PELO_PROVEDOR"), "O Provedor de Pagamento recusou a Tentativa de Pagamento.");
});

test("a frase de sem cancelamento só existe depois que o Pedido saiu para entrega", () => {
  assert.equal(porQueNaoCancela("ENVIADO"), "Este Pedido já saiu para entrega e não pode mais ser cancelado.");
  assert.equal(porQueNaoCancela("ENTREGUE"), "Este Pedido já foi entregue e não pode mais ser cancelado.");
  for (const status of ["AGUARDANDO_PAGAMENTO", "PAGAMENTO_RECUSADO", "PAGO", "SEPARANDO", "CANCELADO", "constructor"]) {
    assert.equal(porQueNaoCancela(status), null, status);
  }
});

test("o botão de cancelar segue o pode_cancelar do Go, e não o Status", () => {
  // A janela é do Go: a tela não redeclara os quatro Status. Um Status da
  // janela com `pode_cancelar` falso não mostra o botão, e o contrário mostra.
  assert.equal(podeCancelar({ status: "SEPARANDO", pode_cancelar: true }), true);
  assert.equal(podeCancelar({ status: "SEPARANDO", pode_cancelar: false }), false);
  assert.equal(podeCancelar({ status: "ENVIADO", pode_cancelar: false }), false);
  assert.equal(podeCancelar({ status: "NOVO", pode_cancelar: true }), true);
  // Resposta sem o campo (um Go mais antigo): sem botão, e não um botão que
  // a rota recusaria.
  assert.equal(podeCancelar({ status: "PAGO" }), false);
});

test("o prazo do cancelamento é o do Confirmar Pedido", async () => {
  const { PRAZO_DA_CONFIRMACAO_MS } = await import(new URL("../lib/checkout.ts", import.meta.url).href);
  assert.equal(PRAZO_DO_CANCELAMENTO_MS, PRAZO_DA_CONFIRMACAO_MS);
});

test("a rota do cancelamento é a do próprio Pedido", () => {
  assert.equal(rotaDoCancelamento("0190-a/b"), "/api/v1/pedidos/0190-a%2Fb/cancelamento");
});

test("a resposta do cancelamento: relê no sucesso e nas corridas, e só a de fora da janela explica", () => {
  const resposta = (status, codigo, mensagem, dados = null) => ({
    resposta: { ok: status < 300, status },
    json: codigo ? { erro: { codigo, mensagem, dados } } : { id: "x", status: "CANCELADO" },
  });
  // O 200 é também o segundo clique, sobre o Pedido já cancelado.
  assert.deepEqual(desfechoDoCancelamento(resposta(200)), { tipo: "releitura", aviso: null });
  assert.deepEqual(desfechoDoCancelamento(resposta(409, "ESTADO_JA_AVANCADO", "m")), { tipo: "releitura", aviso: null });
  // A corrida perdida troca a frase do Status pela que diz o que aconteceu, e
  // não mostra a mensagem crua do Go.
  assert.deepEqual(
    desfechoDoCancelamento(resposta(409, "FORA_DA_JANELA_DE_CANCELAMENTO", "Este Pedido não pode mais ser cancelado.", { status: "ENVIADO" })),
    { tipo: "releitura", aviso: CORRIDA_DO_CANCELAMENTO },
  );
  assert.equal(
    CORRIDA_DO_CANCELAMENTO,
    "Este Pedido foi enviado enquanto você estava nesta tela e não pode mais ser cancelado.",
  );
  assert.deepEqual(desfechoDoCancelamento(resposta(401, "SESSAO_INVALIDA", "m")), { tipo: "semSessao" });
  assert.deepEqual(desfechoDoCancelamento(resposta(404, "NAO_ENCONTRADO", "Recurso não encontrado.")), {
    tipo: "erro",
    mensagem: "Recurso não encontrado.",
  });
  // Um 409 que não é do cancelamento, ou uma resposta sem envelope, diz que
  // falhou em vez de reler calado.
  assert.deepEqual(desfechoDoCancelamento(resposta(409, "TRANSICAO_INVALIDA", "Esta mudança…")), {
    tipo: "erro",
    mensagem: "Esta mudança…",
  });
  assert.deepEqual(desfechoDoCancelamento({ resposta: { ok: false, status: 500 }, json: null }), {
    tipo: "erro",
    mensagem: FALHA_NO_CANCELAMENTO,
  });
});

test("sem o botão, a frase é a da corrida quando houve, e senão a do Status", () => {
  for (const status of ["AGUARDANDO_PAGAMENTO", "PAGAMENTO_RECUSADO", "PAGO", "SEPARANDO"]) {
    assert.equal(fraseSemCancelamento({ status, pode_cancelar: true }, null), null, status);
    // Com o botão na tela, nem a frase da corrida aparece.
    assert.equal(fraseSemCancelamento({ status, pode_cancelar: true }, CORRIDA_DO_CANCELAMENTO), null, status);
  }
  const fora = (status) => ({ status, pode_cancelar: false });
  assert.equal(fraseSemCancelamento(fora("ENVIADO"), null), porQueNaoCancela("ENVIADO"));
  assert.equal(fraseSemCancelamento(fora("ENTREGUE"), null), porQueNaoCancela("ENTREGUE"));
  assert.equal(fraseSemCancelamento(fora("CANCELADO"), null), null);
  assert.equal(fraseSemCancelamento(fora("ENVIADO"), CORRIDA_DO_CANCELAMENTO), CORRIDA_DO_CANCELAMENTO);
});

test("o Dialog nomeia o Pedido pelo número", () => {
  assert.deepEqual(textosDoCancelamento("2026-000042"), {
    botao: "Cancelar Pedido",
    titulo: "Cancelar o Pedido 2026-000042?",
    descricao: "Depois de cancelado, o Pedido não volta a andar.",
    manter: "Manter o Pedido",
    confirmar: "Cancelar Pedido",
    enviando: "Cancelando o Pedido…",
  });
});

test("nenhum texto do cancelamento nomeia a Reserva nem promete reembolso", () => {
  const textos = [
    CORRIDA_DO_CANCELAMENTO,
    FALHA_NO_CANCELAMENTO,
    porQueNaoCancela("ENVIADO"),
    porQueNaoCancela("ENTREGUE"),
    ...Object.values(textosDoCancelamento("2026-000042")),
  ];
  for (const texto of textos) {
    assert.doesNotMatch(texto, /reserva|reembolso|devolu|estorno/i, texto);
  }
});

// ————— O painel de Pedidos do Administrador (6.4) —————

test("o botão nomeia a operação, e o destino desconhecido aparece feio, não invisível", () => {
  assert.equal(rotuloDaAcao("SEPARANDO"), "Iniciar a separação");
  assert.equal(rotuloDaAcao("ENVIADO"), "Registrar o envio");
  assert.equal(rotuloDaAcao("ENTREGUE"), "Registrar a entrega");
  // Uma linha nova na tabela do AD-3 ainda rende um botão legível.
  assert.equal(rotuloDaAcao("DEVOLVIDO"), "Mudar para DEVOLVIDO");
  assert.equal(rotuloDaAcao("CANCELADO"), "Mudar para Cancelado");
  assert.equal(rotaDaTransicao("2026 000042/x"), "/api/v1/admin/pedidos/2026%20000042%2Fx/transicoes");
});

test("a transição inválida é Alert destrutivo, e o Pedido que já avançou é informativo", () => {
  const resposta = (status, codigo, mensagem, dados = null) => ({
    resposta: { ok: status >= 200 && status < 300, status },
    json: { erro: { codigo, mensagem, dados, correlacao: "c" } },
  });
  const tentada = { de: "PAGO", para: "ENTREGUE" };

  // 200: relê, sem Alert nenhum — a releitura traz o Status e o permitidas novos.
  assert.deepEqual(
    desfechoDaTransicao({ resposta: { ok: true, status: 200 }, json: {} }, tentada),
    { tipo: "releitura", alerta: null },
  );

  // Fora da tabela: nomeia a tentada E as permitidas que o Go devolveu.
  const invalida = desfechoDaTransicao(
    resposta(409, "TRANSICAO_INVALIDA", "Esta mudança…", { permitidas: ["SEPARANDO"] }),
    tentada,
  );
  assert.equal(invalida.tipo, "releitura");
  assert.equal(invalida.alerta.variante, "destrutivo");
  assert.equal(
    invalida.alerta.texto,
    "Não é possível ir de Pago para Entregue. A partir de Pago, só: Separando.",
  );
  // Sem saída nenhuma (Pedido terminal) a frase não termina numa lista vazia.
  assert.match(
    desfechoDaTransicao(resposta(409, "TRANSICAO_INVALIDA", "m", { permitidas: [] }), {
      de: "ENTREGUE",
      para: "ENVIADO",
    }).alerta.texto,
    /não tem mudanças de Status disponíveis\.$/,
  );
  // `dados` sem `permitidas` não estoura: cai na mesma frase.
  assert.match(
    desfechoDaTransicao(resposta(409, "TRANSICAO_INVALIDA", "m", {}), tentada).alerta.texto,
    /não tem mudanças de Status disponíveis\.$/,
  );

  // A corrida com a simulação: informativo, com o Status atual, e NUNCA a
  // palavra "inválida" — é ela que reportaria causa falsa (EXPERIENCE).
  const avancou = desfechoDaTransicao(
    resposta(409, "ESTADO_JA_AVANCADO", "O Pedido já avançou de estado.", { status: "SEPARANDO" }),
    { de: "PAGO", para: "SEPARANDO" },
  );
  assert.deepEqual(avancou, {
    tipo: "releitura",
    alerta: { variante: "informativo", texto: "Este Pedido já está em Separando. A linha foi atualizada." },
  });
  assert.doesNotMatch(avancou.alerta.texto, /inválid/i);

  // A Sessão administrativa acabou: o prefixo responde 404, e a tela vai ao login.
  assert.deepEqual(desfechoDaTransicao(resposta(404, "NAO_ENCONTRADO", "m"), tentada), { tipo: "semSessao" });
  // Qualquer outro desfecho é erro de verdade, com a mensagem do envelope.
  assert.deepEqual(desfechoDaTransicao(resposta(400, "CAMPO_INVALIDO", "Campo."), tentada), {
    tipo: "erro",
    mensagem: "Campo.",
  });
  assert.deepEqual(desfechoDaTransicao({ resposta: { ok: false, status: 500 }, json: null }, tentada), {
    tipo: "erro",
    mensagem: FALHA_NA_TRANSICAO,
  });
});

test("o filtro e a ordenação vivem na URL, e o que não vale cai no padrão", () => {
  const url = (busca) => new URLSearchParams(busca);
  assert.deepEqual(filtroDaURL(url("")), { status: "", ordenacao: RECENTES, pagina: 1 });
  assert.deepEqual(filtroDaURL(url("status=PAGO&ordenacao=antigos&pagina=3")), {
    status: "PAGO",
    ordenacao: ANTIGOS,
    pagina: 3,
  });
  // Digitado à mão na barra de endereços: cai no padrão, sem ir ao Go levar 400.
  assert.deepEqual(filtroDaURL(url("status=INVENTADO&ordenacao=preco_asc&pagina=0")), {
    status: "",
    ordenacao: RECENTES,
    pagina: 1,
  });
  assert.equal(filtroDaURL(url("pagina=abc")).pagina, 1);
  assert.equal(filtroDaURL(url("pagina=-9")).pagina, 1);

  // A URL e a chamada saem da mesma montagem, e o padrão não vira ruído.
  assert.equal(consultaDaTabela({ status: "", ordenacao: RECENTES, pagina: 1 }), "");
  assert.equal(enderecoDaTabela({ status: "", ordenacao: RECENTES, pagina: 1 }), "/admin/pedidos");
  assert.equal(rotaDaTabela({ status: "", ordenacao: RECENTES, pagina: 1 }), "/api/v1/admin/pedidos");
  const f = { status: "PAGO", ordenacao: ANTIGOS, pagina: 2 };
  assert.equal(consultaDaTabela(f), "status=PAGO&ordenacao=antigos&pagina=2");
  assert.equal(enderecoDaTabela(f), "/admin/pedidos?status=PAGO&ordenacao=antigos&pagina=2");
  assert.equal(rotaDaTabela(f), "/api/v1/admin/pedidos?status=PAGO&ordenacao=antigos&pagina=2");
  // E a volta fecha: o que a URL montou é o que ela lê.
  assert.deepEqual(filtroDaURL(url(consultaDaTabela(f))), f);
});

test("o filtro oferece os sete Status do ponto único de rótulo, e as duas ordenações", () => {
  assert.equal(STATUS.length, 7);
  for (const s of STATUS) assert.notEqual(rotuloDoStatus(s), s, s);
  assert.deepEqual(ORDENACOES, [RECENTES, ANTIGOS]);
});

test("a Tabela consulta em intervalo só enquanto listar Pedido não terminal", () => {
  assert.equal(intervaloDaTabela([]), null);
  assert.equal(intervaloDaTabela([{ terminal: true }, { terminal: true }]), null);
  assert.equal(intervaloDaTabela([{ terminal: true }, { terminal: false }]), INTERVALO_AVANCANDO_MS);
});

test("a Tabela anuncia só quem mudou de Status, nomeando o Pedido", () => {
  const linha = (id, numero, status) => ({ id, numero, status });
  const antes = [linha("a", "2026-000001", "PAGO"), linha("b", "2026-000002", "SEPARANDO")];
  const mapa = statusPorPedido(antes);
  assert.deepEqual(mapa, { a: "PAGO", b: "SEPARANDO" });

  // Nada mudou: nada a anunciar — o leitor de tela não relê o que já leu.
  assert.equal(anuncioDaTabela(mapa, antes), "");
  // Um mudou: o anúncio nomeia o Pedido, e não só o Status.
  assert.equal(
    anuncioDaTabela(mapa, [linha("a", "2026-000001", "SEPARANDO"), antes[1]]),
    "Pedido 2026-000001: Separando.",
  );
  // Dois mudaram: uma frase por Pedido, na mesma região.
  assert.equal(
    anuncioDaTabela(mapa, [linha("a", "2026-000001", "SEPARANDO"), linha("b", "2026-000002", "ENVIADO")]),
    "Pedido 2026-000001: Separando. Pedido 2026-000002: Enviado.",
  );
  // Pedido que ainda não estava na leitura anterior é chegada, e não mudança.
  assert.equal(anuncioDaTabela(mapa, [...antes, linha("c", "2026-000003", "PAGO")]), "");
  // A primeira leitura não anuncia a lista inteira.
  assert.equal(anuncioDaTabela({}, antes), "");
  // Chave herdada de Object.prototype não conta como Status anterior.
  assert.equal(anuncioDaTabela({}, [linha("constructor", "2026-000004", "PAGO")]), "");
});
