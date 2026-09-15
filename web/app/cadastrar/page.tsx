"use client";

import { useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

// Tela de cadastro da estória 2.1. Nenhuma regra mora aqui (AD-10): a defesa
// é o Go, e quem decide é sempre o 400 dele. O `required` e o `type="email"`
// não chegam a barrar envio nenhum — o `noValidate` do formulário desliga a
// validação nativa —, e ficam pela semântica: `aria-required` para o leitor
// de tela e o teclado certo no celular.
//
// O caminho é relativo (`/api/...`) de propósito: chamado do navegador, é a
// mesma origem, e só assim o Set-Cookie volta como cookie de primeira parte.
//
// O contrato do erro em linha é `dados.campo` (UX-DR16): o envelope diz de
// qual campo a mensagem é, e a tela liga os dois por `aria-describedby` e põe
// o foco no campo. Vale igual para o 400 de limite e para o 409 do e-mail
// repetido — uma leitura só.

type ErroDeCampo = { campo: Campo | null; mensagem: string; duplicado: boolean };

const CAMPOS = ["nome", "email", "senha"] as const;
type Campo = (typeof CAMPOS)[number];

export default function Cadastrar() {
  const router = useRouter();
  const [valores, setValores] = useState<Record<Campo, string>>({ nome: "", email: "", senha: "" });
  const [erro, setErro] = useState<ErroDeCampo | null>(null);
  const [enviando, setEnviando] = useState(false);

  // Um ref por campo para o foco ir ao campo recusado ao submeter (UX-DR20c).
  const refs: Record<Campo, React.RefObject<HTMLInputElement | null>> = {
    nome: useRef<HTMLInputElement>(null),
    email: useRef<HTMLInputElement>(null),
    senha: useRef<HTMLInputElement>(null),
  };

  function recusar(erroDeCampo: ErroDeCampo) {
    setErro(erroDeCampo);
    if (erroDeCampo.campo) refs[erroDeCampo.campo].current?.focus();
  }

  async function cadastrar(evento: React.FormEvent) {
    evento.preventDefault();
    setEnviando(true);
    setErro(null);
    try {
      const resposta = await fetch("/api/v1/compradores", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(valores),
      });
      const corpo = await resposta.json().catch(() => null);
      if (!resposta.ok) {
        // A mensagem exibida é sempre a do envelope do servidor (AD-14),
        // nunca uma inventada aqui — é ela que fala a Voice and Tone da UX e
        // que nomeia o limiar, que mora na Config e não no JavaScript.
        // O campo é filtrado contra CAMPOS: um nome que a tela não conhece
        // não casaria em nenhum campo nem no alerta genérico, e a mensagem do
        // servidor sumiria da tela inteira.
        recusar({
          campo: CAMPOS.find((c) => c === corpo?.erro?.dados?.campo) ?? null,
          mensagem: corpo?.erro?.mensagem ?? "Não foi possível criar a conta.",
          duplicado: corpo?.erro?.codigo === "EMAIL_JA_CADASTRADO",
        });
        setEnviando(false);
        return;
      }
      // O cadastro já abriu a Sessão: o visitante chega à Vitrine autenticado.
      // O botão continua desabilitado durante a navegação — reabilitá-lo abriria
      // a janela para um segundo envio enquanto a tela ainda é esta.
      router.push("/");
    } catch {
      recusar({ campo: null, mensagem: "Não foi possível falar com o servidor.", duplicado: false });
      setEnviando(false);
    }
  }

  function campoDe(
    campo: Campo,
    rotulo: string,
    tipo: string,
    autoComplete: string,
    dica?: string,
  ) {
    const comErro = erro?.campo === campo;
    const idErro = `${campo}-erro`;
    const idDica = `${campo}-dica`;
    // O aria-describedby aponta para a dica e, quando há, para o erro: quem
    // usa leitor de tela ouve os dois ao chegar no campo.
    const descrito = [dica ? idDica : null, comErro ? idErro : null].filter(Boolean).join(" ");
    return (
      <div className="space-y-2">
        <Label htmlFor={campo}>{rotulo}</Label>
        <Input
          id={campo}
          name={campo}
          type={tipo}
          autoComplete={autoComplete}
          required
          ref={refs[campo]}
          aria-invalid={comErro || undefined}
          aria-describedby={descrito || undefined}
          value={valores[campo]}
          onChange={(e) => setValores({ ...valores, [campo]: e.target.value })}
        />
        {dica && (
          <p id={idDica} className="text-muted-foreground text-sm">
            {dica}
          </p>
        )}
        {comErro && (
          <p id={idErro} className="text-destructive text-sm" role="alert">
            {erro.mensagem}
            {erro.duplicado && (
              <>
                {" "}
                <a className="text-link underline" href="/entrar">
                  Entrar na conta existente
                </a>
              </>
            )}
          </p>
        )}
      </div>
    );
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
              <h1>Criar conta</h1>
            </CardTitle>
          </CardHeader>
          <CardContent>
            {/* noValidate: a mensagem que vale é a do Go, e a bolha nativa do
                navegador a esconderia antes de a requisição sair. */}
            <form className="space-y-4" onSubmit={cadastrar} noValidate>
              {campoDe("nome", "Nome", "text", "name")}
              {campoDe("email", "E-mail", "email", "username")}
              {campoDe("senha", "Senha", "password", "new-password", "Pelo menos 8 caracteres.")}

              {/* Erro sem campo — rede fora, ou falha que o servidor não
                  atribuiu a um campo. */}
              {erro && !erro.campo && (
                <Alert variant="destructive">
                  <AlertDescription>{erro.mensagem}</AlertDescription>
                </Alert>
              )}

              <Button type="submit" disabled={enviando}>
                {enviando ? "Criando conta…" : "Criar conta"}
              </Button>
            </form>

            <p className="text-muted-foreground mt-4 text-sm">
              Já tem conta?{" "}
              <a className="text-link underline" href="/entrar">
                Entrar
              </a>
            </p>
          </CardContent>
        </Card>
      </main>
    </div>
  );
}
