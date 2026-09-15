// Cobre a linha "destino forjado" da matriz de I/O da estória 2.2: o que a tela
// de Login aceita como ponto de retorno. O módulo é `.ts` puro justamente para
// este teste poder importá-lo — o Node 24 lê tipo direto, mas não JSX.
import { test } from "node:test";
import assert from "node:assert/strict";

const { destinoSeguro, paraLogin } = await import(
  new URL("../lib/destino.ts", import.meta.url).href
);

test("destino: caminho relativo passa inteiro, com busca e âncora", () => {
  for (const destino of ["/pedidos/abc", "/", "/produtos/1?x=2", "/a#b"]) {
    assert.equal(destinoSeguro(`?destino=${encodeURIComponent(destino)}`), destino);
  }
});

test("destino: absoluto, protocolo-relativo e barra invertida caem em /", () => {
  const forjados = [
    "http://mal",
    "https://mal/entrar",
    ["//", "mal"].join(""),
    "/\\mal",
    "javascript:alert(1)",
    "mal.com",
  ];
  for (const destino of forjados) {
    assert.equal(
      destinoSeguro(`?destino=${encodeURIComponent(destino)}`),
      "/",
      `${destino} foi aceito como destino`,
    );
  }
});

// O navegador descarta tabulação, CR e LF antes de resolver a URL: sem esta
// recusa, `/%09/mal.com` passa pelo guarda da barra e vira `//mal.com` na hora
// de navegar — o redirecionador aberto que a barra invertida fecha.
test("destino: caractere de controle codificado cai em /", () => {
  for (const codigo of ["%09", "%0a", "%0d", "%00"]) {
    const destino = `/${codigo}/mal.com`;
    assert.equal(destinoSeguro(`?destino=${destino}`), "/", `${destino} foi aceito`);
  }
});

// Sem isto a página empurra para si mesma e o botão fica preso em "Entrando…".
test("destino: o próprio /entrar cai em /", () => {
  for (const destino of ["/entrar", "/entrar?destino=/a", "/entrar#x"]) {
    assert.equal(destinoSeguro(`?destino=${encodeURIComponent(destino)}`), "/", `${destino} foi aceito`);
  }
  // Mas um caminho que apenas começa por "entrar" não é a tela de Login.
  assert.equal(destinoSeguro("?destino=%2Fentrarios"), "/entrarios");
});

// A ida e volta: o que paraLogin monta é o que destinoSeguro devolve inteiro —
// incluindo a busca, que é metade do "ponto em que o Comprador parou".
test("destino: paraLogin e destinoSeguro fecham a volta", () => {
  for (const caminho of ["/pedidos/abc", "/produtos/1?cor=azul&n=2"]) {
    const link = paraLogin(caminho);
    assert.equal(destinoSeguro(new URL(link, "http://azamon.test").search), caminho);
  }
});

test("destino: ausente ou vazio vira /", () => {
  assert.equal(destinoSeguro(""), "/");
  assert.equal(destinoSeguro("?outra=coisa"), "/");
  assert.equal(destinoSeguro("?destino="), "/");
});
