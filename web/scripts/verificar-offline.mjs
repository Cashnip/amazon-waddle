// O NFR-15 exige percorrer o Roteiro A com a rede desconectada. Fonte, CSS ou
// script vindos de fora falham em silêncio — e falham na sala, na apresentação.
// O AD-12 manda um grep pelas quatro formas de escrever isso sem perceber, e
// manda que ele **derrube o build**. Roda como `prebuild`, então `npm run build`
// e a construção da imagem `web` param juntos, sem infraestrutura nova.
//
// De carona vai a outra armadilha do Next 16 que falha calada: um arquivo
// chamado `middleware.ts` é ignorado sem erro de build. A estória 1.2 a trata
// por omissão — daqui em diante o arquivo não pode voltar sem alguém notar.
import { readdirSync, readFileSync } from "node:fs";
import { join, relative, extname, dirname } from "node:path";
import { fileURLToPath } from "node:url";

// Sem argumento, varre o `web/` inteiro — que é como o `prebuild` o chama.
// O argumento existe para o teste apontar a varredura para uma pasta de exemplo.
const raiz = process.argv[2] ?? dirname(dirname(fileURLToPath(import.meta.url)));
const PADROES = ["fonts.googleapis", "@import url(http", "//cdn.", "next/font/google"];
const PROIBIDOS = new Set(["middleware.ts", "middleware.js"]);
// O próprio verificador é o único lugar onde os padrões podem aparecer.
const IGNORADOS = new Set(["node_modules", ".next", "package-lock.json", "verificar-offline.mjs"]);
// Lista de negação, não de permissão: extensão desconhecida é lida, não pulada.
// Falha fechada — quem inventar um `.vue` ou um arquivo sem extensão entra na
// varredura em vez de escapar dela em silêncio.
const BINARIOS = new Set([
  ".woff", ".woff2", ".ttf", ".otf", ".eot",
  ".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif", ".ico", ".bmp",
  ".pdf", ".zip", ".gz", ".mp4", ".webm", ".mp3", ".wasm",
]);

const achados = [];

function varrer(dir) {
  for (const item of readdirSync(dir, { withFileTypes: true })) {
    if (IGNORADOS.has(item.name)) continue;
    const caminho = join(dir, item.name);
    if (item.isDirectory()) {
      varrer(caminho);
      continue;
    }
    if (PROIBIDOS.has(item.name.toLowerCase())) {
      achados.push(`${relative(raiz, caminho)}: arquivo proibido — o Next 16 ignora ${item.name} sem erro; o nome é proxy.ts`);
    }
    if (BINARIOS.has(extname(item.name).toLowerCase())) continue;
    const linhas = readFileSync(caminho, "utf8").split(/\r?\n/);
    for (const [i, linha] of linhas.entries()) {
      const alvo = linha.toLowerCase();
      for (const padrao of PADROES) {
        if (alvo.includes(padrao)) {
          achados.push(`${relative(raiz, caminho)}:${i + 1}: ${padrao} — ${linha.trim()}`);
        }
      }
    }
  }
}

varrer(raiz);

if (achados.length > 0) {
  console.error("Recurso de rede externa ou armadilha do Next no web/ (AD-12, NFR-15):");
  for (const a of achados) console.error(`  ${a}`);
  process.exit(1);
}
console.log(`verificar-offline: nenhum dos ${PADROES.length} padrões encontrado.`);
