"use client";

import { useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

// Nenhuma regra mora aqui (AD-10): quem recusa a senha curta é o 400 do Go, e a
// mensagem exibida é sempre a do envelope (AD-14) — é ela que nomeia o limiar,
// que vive na Config.
//
// O contrato do erro em linha é `dados.campo` (UX-DR20c), como no cadastro: o
// envelope diz de qual campo a mensagem é, a tela liga os dois por
// `aria-describedby` e põe o foco no campo.
//
// O 404 TOKEN_INVALIDO é o outro estado da tela, e não um erro de campo: o link
// não vale mais, e o que resta a fazer é pedir outro.

// O erro carrega o campo que o envelope nomeou: sem campo, é falha de rede ou
// do servidor, e vai para o alerta geral em vez de acusar a senha digitada.
type ErroDeCampo = { campo: "senha" | null; mensagem: string };

export function Redefinir({ token }: { token: string }) {
  const router = useRouter();
  const [senha, setSenha] = useState("");
  const [erro, setErro] = useState<ErroDeCampo | null>(null);
  const [expirado, setExpirado] = useState<string | null>(null);
  const [enviando, setEnviando] = useState(false);
  const campo = useRef<HTMLInputElement>(null);

  async function redefinir(evento: React.FormEvent) {
    evento.preventDefault();
    setEnviando(true);
    setErro(null);
    try {
      const resposta = await fetch(
        `/api/v1/redefinicoes-de-senha/${encodeURIComponent(token)}`,
        {
          method: "PUT",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ senha }),
        },
      );
      if (!resposta.ok) {
        const corpo = await resposta.json().catch(() => null);
        if (corpo?.erro?.codigo === "TOKEN_INVALIDO") {
          // A mensagem é a do envelope (AD-14), como em toda a casca: é ela que
          // fala a Voice and Tone da UX, e uma cópia aqui divergiria dela.
          setExpirado(corpo?.erro?.mensagem ?? "Este link de redefinição não vale mais.");
          setEnviando(false);
          return;
        }
        const doCampo = corpo?.erro?.dados?.campo === "senha";
        setErro({
          campo: doCampo ? "senha" : null,
          mensagem: corpo?.erro?.mensagem ?? "Não foi possível redefinir a senha.",
        });
        if (doCampo) campo.current?.focus();
        setEnviando(false);
        return;
      }
      // A redefinição encerrou TODAS as Sessões e não abriu nenhuma: o caminho
      // de volta é o Login, com a senha nova. O botão continua desabilitado
      // durante a navegação, como no cadastro.
      router.push("/entrar");
    } catch {
      setErro({ campo: null, mensagem: "Não foi possível falar com o servidor." });
      setEnviando(false);
    }
  }

  if (expirado) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>
            <h1>Link expirado</h1>
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4" role="status">
          <p>{expirado}</p>
          <Button asChild>
            <a href="/esqueci-a-senha">Pedir outro link</a>
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          <h1>Criar senha nova</h1>
        </CardTitle>
      </CardHeader>
      <CardContent>
        {/* noValidate: a mensagem que vale é a do Go, e a bolha nativa do
            navegador a esconderia antes de a requisição sair. */}
        <form className="space-y-4" onSubmit={redefinir} noValidate>
          <div className="space-y-2">
            <Label htmlFor="senha">Senha nova</Label>
            <Input
              id="senha"
              name="senha"
              type="password"
              autoComplete="new-password"
              required
              autoFocus
              ref={campo}
              aria-invalid={erro?.campo === "senha" || undefined}
              aria-describedby={["senha-dica", erro?.campo === "senha" ? "senha-erro" : null]
                .filter(Boolean)
                .join(" ")}
              value={senha}
              onChange={(e) => setSenha(e.target.value)}
            />
            <p id="senha-dica" className="text-muted-foreground text-sm">
              Pelo menos 8 caracteres.
            </p>
            {erro?.campo === "senha" && (
              <p id="senha-erro" className="text-destructive text-sm" role="alert">
                {erro.mensagem}
              </p>
            )}
          </div>

          {/* Erro sem campo — rede fora, ou falha que o servidor não atribuiu
              a um campo. */}
          {erro && !erro.campo && (
            <Alert variant="destructive">
              <AlertDescription>{erro.mensagem}</AlertDescription>
            </Alert>
          )}

          {/* Aviso estático, e não <Alert>: o componente fixa role="alert", e
              um aviso que não mudou seria anunciado como interrupção assim que
              a tela abre, na frente do rótulo do campo. */}
          <p className="text-muted-foreground text-sm">
            Ao redefinir, todas as sessões abertas nesta conta são encerradas.
          </p>

          <Button type="submit" disabled={enviando}>
            {enviando ? "Redefinindo…" : "Redefinir senha"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
