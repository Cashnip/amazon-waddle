// Cobre as linhas da matriz de I/O da estória 1.2 que vivem no lado do Next:
// o rewrite de /api e as duas faces da guarda offline.
//
// Os padrões proibidos nunca aparecem literalmente neste arquivo — ele mora
// dentro de `web/`, e a própria guarda o varre. Por isso são montados por
// concatenação.
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { join, dirname } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";

const aqui = dirname(fileURLToPath(import.meta.url));
const verificador = join(aqui, "verificar-offline.mjs");

// A mesma lista que a guarda carrega. Se ela perder um padrão, o laço abaixo
// deixa de encontrá-lo e o teste cai — que é a única forma de a lista não
// encolher em silêncio.
const PADROES = [
  ["fonts.", "googleapis"].join(""),
  ["@import ", "url(http"].join(""),
  ["//", "cdn."].join(""),
  ["next/", "font/google"].join(""),
];

function verificar(raiz) {
  return spawnSync(process.execPath, [verificador, raiz], { encoding: "utf8" });
}

function pastaCom(arquivo, conteudo) {
  const dir = mkdtempSync(join(tmpdir(), "azamon-guarda-"));
  writeFileSync(join(dir, arquivo), conteudo);
  return dir;
}

// O rewrite é lido do módulo, e o módulo lê o ambiente no arranque — então
// cada caso precisa de uma instância nova. A busca na URL só existe para
// furar o cache de módulo do Node.
let versao = 0;
async function carregarConfig(valor) {
  if (valor === undefined) delete process.env.AZAMON_API_URL;
  else process.env.AZAMON_API_URL = valor;
  const url = new URL("../next.config.ts", import.meta.url);
  url.search = `v=${versao++}`;
  return (await import(url.href)).default;
}

test("rewrite: /api/* vai para o Go, e nada mais é reescrito", async () => {
  const config = await carregarConfig(undefined);
  const regras = await config.rewrites();

  assert.equal(regras.length, 1, "uma regra só: /api");
  assert.equal(regras[0].source, "/api/:caminho*");
  assert.equal(regras[0].destination, "http://azamon:8080/api/:caminho*");
  // A recusa do next/image a upstream de IP privado falha calada (AD-10).
  assert.equal(config.images?.unoptimized, true);
});

test("rewrite: AZAMON_API_URL redireciona o destino", async () => {
  const config = await carregarConfig("http://localhost:8080");
  const [regra] = await config.rewrites();
  assert.equal(regra.destination, "http://localhost:8080/api/:caminho*");
});

test("rewrite: valor vazio ou com barra sobrando volta ao padrão do compose", async () => {
  for (const valor of ["", "http://azamon:8080/", "http://azamon:8080///"]) {
    const config = await carregarConfig(valor);
    const [regra] = await config.rewrites();
    assert.equal(
      regra.destination,
      "http://azamon:8080/api/:caminho*",
      `AZAMON_API_URL=${JSON.stringify(valor)} produziu ${regra.destination}`,
    );
  }
});

test("guarda offline: cada padrão proibido derruba o build, em qualquer caixa", () => {
  for (const padrao of PADROES) {
    for (const escrito of [padrao, padrao.toUpperCase()]) {
      const dir = pastaCom("estilo.css", `algo ${escrito} algo\n`);
      try {
        const r = verificar(dir);
        assert.equal(r.status, 1, `${escrito} passou batido`);
        assert.match(r.stderr, /estilo\.css:1:/);
        assert.ok(r.stderr.includes(padrao), r.stderr);
      } finally {
        rmSync(dir, { recursive: true, force: true });
      }
    }
  }
});

test("guarda offline: middleware.ts de volta é achado — o Next 16 o ignora calado", () => {
  const dir = pastaCom("middleware.ts", "export function middleware() {}\n");
  try {
    const r = verificar(dir);
    assert.equal(r.status, 1, "middleware.ts passou batido");
    assert.match(r.stderr, /middleware\.ts/);
    assert.match(r.stderr, /proxy\.ts/);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test("guarda offline: só fonte local e CSS próprio deixa o build seguir", () => {
  const dir = pastaCom("estilo.css", `@font-face { src: url("/fonts/azamon-sans.woff2"); }\n`);
  try {
    const r = verificar(dir);
    assert.equal(r.status, 0, r.stderr);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test("guarda offline: o web/ como está hoje passa", () => {
  assert.equal(verificar(dirname(aqui)).status, 0);
});
