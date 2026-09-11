import type { NextConfig } from "next";

// Um caminho de dado só: navegador → /api → Go (AD-10). O navegador conhece
// uma origem só (:3000), então não há CORS nem SameSite=None, e o Set-Cookie
// do Go volta como cookie de primeira parte.
// `??` só cobre null/undefined: string vazia viraria destino "/api/:caminho*",
// que é o Next reescrevendo para si mesmo em laço. Barra à direita viraria
// "//api/...", que o navegador lê como outro host.
const api =
  (process.env.AZAMON_API_URL || "http://azamon:8080").replace(/\/+$/, "");

const nextConfig: NextConfig = {
  async rewrites() {
    return [{ source: "/api/:caminho*", destination: `${api}/api/:caminho*` }];
  },
  // O Next 16 recusa otimizar imagem cujo upstream resolva para IP privado
  // — em compose, o Go é exatamente isso. Desligado antes que exista imagem,
  // porque a falha é um 400 silencioso na sala (AD-12).
  images: { unoptimized: true },
};

export default nextConfig;
