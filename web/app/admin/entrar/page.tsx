"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

// O login da área administrativa, no molde de `/entrar` e sem a `Casca` da
// loja: nem busca, nem Carrinho, nem Menu da conta. A rota do Go consulta só
// `identidade.administrador` — o Comprador leva o mesmo CREDENCIAL_INVALIDA
// de quem não existe, e a mensagem é sempre a do envelope (AD-14).
export default function EntrarAdministrador() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [senha, setSenha] = useState("");
  const [erro, setErro] = useState<string | null>(null);
  const [enviando, setEnviando] = useState(false);

  async function entrar(evento: React.FormEvent) {
    evento.preventDefault();
    setEnviando(true);
    setErro(null);
    try {
      const resposta = await fetch("/api/v1/admin/sessoes", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, senha }),
      });
      if (!resposta.ok) {
        const corpo = await resposta.json().catch(() => null);
        setErro(corpo?.erro?.mensagem ?? "Não foi possível entrar.");
        setEnviando(false);
        return;
      }
      router.push("/admin/vendedores");
    } catch {
      setErro("Não foi possível falar com o servidor.");
      setEnviando(false);
    }
  }

  return (
    <div className="min-h-svh">
      <header className="bg-chrome-muted text-chrome-foreground">
        <div className="mx-auto flex max-w-conteudo items-center px-page-margin py-4 lg:px-page-margin-lg">
          <span className="wordmark">azamon</span>
        </div>
      </header>
      <main className="mx-auto max-w-md px-page-margin py-section-sm">
        <Card>
          <CardHeader>
            <CardTitle>
              <h1>Entrar como Administrador</h1>
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
