"use client";

import { useState } from "react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Casca } from "@/components/casca";

// Tela de solicitação da estória 2.3 (UX-DR20a). O estado de sucesso é neutro
// e aparece SEMPRE: o servidor responde o mesmo 202 exista ou não a conta, e
// uma tela que dissesse "não encontramos esse e-mail" desfaria do lado do
// navegador o que o Go fez questão de esconder.
//
// Não há serviço de e-mail (AD-20): na demonstração o caminho sai no log do
// servidor — `docker compose logs azamon | grep redefinir-senha`. O texto não
// promete inbox nenhuma por isso.

export default function EsqueciASenha() {
  const [email, setEmail] = useState("");
  const [enviado, setEnviado] = useState(false);
  const [erro, setErro] = useState<string | null>(null);
  const [enviando, setEnviando] = useState(false);

  async function solicitar(evento: React.FormEvent) {
    evento.preventDefault();
    setEnviando(true);
    setErro(null);
    try {
      const resposta = await fetch("/api/v1/redefinicoes-de-senha", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
      });
      // Só a rede falha aqui: a rota responde 202 para toda entrada, e um
      // status fora disso é defeito do servidor, não resposta sobre a conta.
      if (!resposta.ok) {
        const corpo = await resposta.json().catch(() => null);
        setErro(corpo?.erro?.mensagem ?? "Não foi possível enviar a solicitação.");
        setEnviando(false);
        return;
      }
      setEnviado(true);
    } catch {
      setErro("Não foi possível falar com o servidor.");
    }
    setEnviando(false);
  }

  return (
    <Casca>
      <main className="mx-auto max-w-md px-page-margin py-section-sm">
        <Card>
          <CardHeader>
            <CardTitle>
              <h1>Esqueci minha senha</h1>
            </CardTitle>
          </CardHeader>
          <CardContent>
            {enviado ? (
              // role="status" para o leitor de tela anunciar a troca sem o
              // foco precisar ir a lugar nenhum.
              <div className="space-y-4" role="status">
                <p>
                  Solicitação registrada. Se existir uma conta com esse e-mail, o link de
                  redefinição já foi emitido e vale por tempo limitado.
                </p>
                <p className="text-muted-foreground text-sm">
                  Digitou o endereço errado? Peça de novo.
                </p>
                <Button variant="outline" onClick={() => setEnviado(false)}>
                  Usar outro e-mail
                </Button>
              </div>
            ) : (
              <>
                {/* noValidate: a mensagem que vale é a do Go, como no cadastro. */}
                <form className="space-y-4" onSubmit={solicitar} noValidate>
                  <div className="space-y-2">
                    <Label htmlFor="email">E-mail</Label>
                    <Input
                      id="email"
                      name="email"
                      type="email"
                      autoComplete="username"
                      required
                      autoFocus
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                    />
                  </div>

                  {erro && (
                    <Alert variant="destructive">
                      <AlertDescription>{erro}</AlertDescription>
                    </Alert>
                  )}

                  <Button type="submit" disabled={enviando}>
                    {enviando ? "Enviando…" : "Enviar link de redefinição"}
                  </Button>
                </form>

                <p className="text-muted-foreground mt-4 text-sm">
                  Lembrou a senha?{" "}
                  <a className="text-link underline" href="/entrar">
                    Entrar
                  </a>
                </p>
              </>
            )}
          </CardContent>
        </Card>
      </main>
    </Casca>
  );
}
