"use client";

import { useState } from "react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

// Tela crua da estória 1.5: o que ela prova é que o cookie de Sessão sai do Go
// e atravessa o rewrites() do Next íntegro. A composição de marca do login
// nasce na Épica 2, que é dona de identidade.
//
// O caminho é relativo (`/api/...`) de propósito: chamado do navegador, é a
// mesma origem, e só assim o Set-Cookie volta como cookie de primeira parte.
// Apontar para http://azamon:8080 daqui perderia o cookie em silêncio.

export default function Entrar() {
  const [email, setEmail] = useState("");
  const [senha, setSenha] = useState("");
  const [nome, setNome] = useState<string | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [enviando, setEnviando] = useState(false);

  async function entrar(evento: React.FormEvent) {
    evento.preventDefault();
    setEnviando(true);
    setErro(null);
    setNome(null);
    try {
      const resposta = await fetch("/api/v1/sessoes", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, senha }),
      });
      const corpo = await resposta.json();
      // A mensagem exibida é a do envelope do servidor (AD-14), nunca uma
      // inventada aqui — é ela que fala a Voice and Tone da UX.
      if (!resposta.ok) setErro(corpo?.erro?.mensagem ?? "Não foi possível entrar.");
      else setNome(corpo.nome);
    } catch {
      setErro("Não foi possível falar com o servidor.");
    } finally {
      setEnviando(false);
    }
  }

  return (
    <div className="min-h-svh">
      <header className="bg-chrome text-chrome-foreground">
        <div className="mx-auto flex max-w-conteudo items-center px-page-margin py-4 lg:px-page-margin-lg">
          <a className="wordmark" href="/">
            azamon
          </a>
        </div>
      </header>

      <main className="mx-auto max-w-md px-page-margin py-section-sm">
        <Card>
          <CardHeader>
            <CardTitle>
              <h1>Entrar</h1>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <form className="space-y-4" onSubmit={entrar}>
              <div className="space-y-2">
                <Label htmlFor="email">E-mail</Label>
                <Input
                  id="email"
                  name="email"
                  type="email"
                  autoComplete="username"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="senha">Senha</Label>
                <Input
                  id="senha"
                  name="senha"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={senha}
                  onChange={(e) => setSenha(e.target.value)}
                />
              </div>

              {erro && (
                <Alert variant="destructive">
                  <AlertDescription>{erro}</AlertDescription>
                </Alert>
              )}
              {nome && (
                <Alert>
                  <AlertDescription>
                    Sessão aberta. Olá, {nome}.{" "}
                    <a className="text-link underline" href="/">
                      Voltar à conferência
                    </a>
                  </AlertDescription>
                </Alert>
              )}

              <Button type="submit" disabled={enviando}>
                {enviando ? "Entrando…" : "Entrar"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </main>
    </div>
  );
}
