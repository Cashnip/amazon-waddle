"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { paraLogin } from "@/lib/destino";
import { sair } from "@/lib/sessao";

// Perfil (2.6): o e-mail em leitura — a Sessão já carrega, e uma rota
// separada pagaria uma ida a mais ao Postgres pelo mesmo dado. "Trocar senha"
// leva ao `/esqueci-a-senha` existente (FR-3): nenhuma FR pede formulário de
// troca com a senha atual, e reescrever aquele fluxo aqui duplicaria o que já
// existe. Sem Sessão, redireciona a `/entrar` preservando o destino — o mesmo
// molde de `meus-enderecos.tsx`.
export function Perfil() {
  const router = useRouter();
  const [email, setEmail] = useState<string | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const carregar = useCallback(async () => {
    try {
      const resposta = await fetch("/api/v1/sessao", { credentials: "same-origin" });
      if (resposta.status === 401) {
        router.push(paraLogin());
        return;
      }
      const corpo = await resposta.json().catch(() => null);
      if (!resposta.ok) {
        setErro(corpo?.erro?.mensagem ?? "Não foi possível ler o Perfil.");
        return;
      }
      setErro(null);
      setEmail(corpo?.email ?? null);
    } catch {
      setErro("Não foi possível falar com o servidor.");
    }
  }, [router]);

  useEffect(() => {
    carregar();
  }, [carregar]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          <h1>Perfil</h1>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {erro && (
          <Alert variant="destructive">
            <AlertDescription>{erro}</AlertDescription>
          </Alert>
        )}

        <div className="space-y-1">
          <p className="text-muted-foreground text-sm">E-mail</p>
          {/* "…" enquanto a primeira leitura está a caminho: afirmar um
              e-mail vazio antes da resposta chegar seria uma afirmação falsa
              sobre a conta. */}
          <p>{email ?? "…"}</p>
        </div>

        <a className="text-link block text-sm underline" href="/esqueci-a-senha">
          Trocar senha
        </a>

        <Button variant="outline" onClick={sair}>
          Sair
        </Button>
      </CardContent>
    </Card>
  );
}
