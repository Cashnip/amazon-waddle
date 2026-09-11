import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Azamon",
};

// A família vem do @font-face de globals.css, apontando para um WOFF2 do
// próprio repositório. Nada de fonte remota: o NFR-15 manda percorrer o
// Roteiro A com a rede desconectada, e recurso externo falha calado, na sala.
// O que não pode aparecer aqui está listado em scripts/verificar-offline.mjs.
export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="pt-BR" className="font-sans">
      <body>{children}</body>
    </html>
  );
}
