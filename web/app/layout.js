// A casca visual — fonte auto-hospedada, tokens e shadcn — é da estória 1.2.
// Aqui a página existe só para o contêiner `web` subir.
export const metadata = {
  title: "Azamon",
};

export default function RootLayout({ children }) {
  return (
    <html lang="pt-BR">
      <body>{children}</body>
    </html>
  );
}
