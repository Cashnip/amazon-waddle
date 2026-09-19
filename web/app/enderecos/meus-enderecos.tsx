"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { EnderecoPorExtenso, FormularioDeEndereco } from "@/components/formulario-de-endereco";
import { paraLogin } from "@/lib/destino";
import type { Endereco } from "@/lib/endereco";

// Nenhuma regra mora aqui (AD-10): a defesa é o Go. O formulário — erro em
// linha por `dados.campo`, foco no campo, 409 do teto no alerta — é o mesmo do
// checkout, e mora em `FormularioDeEndereco` desde a 5.2.

export function MeusEnderecos() {
  const router = useRouter();
  const [enderecos, setEnderecos] = useState<Endereco[] | null>(null);
  const [erroDaLista, setErroDaLista] = useState<string | null>(null);
  // `null` = nenhum formulário aberto; "" = cadastrando; um id = editando.
  const [editando, setEditando] = useState<string | null>(null);
  // Conta as aberturas: abrir de novo o mesmo Endereço refaz o formulário, como
  // sempre refez, em vez de manter o que estava digitado.
  const [aberturas, setAberturas] = useState(0);
  const [enviando, setEnviando] = useState(false);
  const [aRemover, setARemover] = useState<Endereco | null>(null);

  // Sessão expirada leva ao Login com o caminho atual no `destino`: a tela não
  // tem o que mostrar sem Sessão, e o Comprador volta a esta mesma lista depois
  // de entrar.
  const semSessao = useCallback(() => router.push(paraLogin()), [router]);

  const carregar = useCallback(async () => {
    try {
      const resposta = await fetch("/api/v1/enderecos", { credentials: "same-origin" });
      if (resposta.status === 401) {
        semSessao();
        return;
      }
      const corpo = await resposta.json().catch(() => null);
      if (!resposta.ok) {
        setErroDaLista(corpo?.erro?.mensagem ?? "Não foi possível ler os Endereços.");
        return;
      }
      setErroDaLista(null);
      const lista: Endereco[] = corpo ?? [];
      setEnderecos(lista);
      // O Endereço em edição saiu da lista (removido com o formulário aberto):
      // o formulário fecha. Aberto, ele perderia o Endereço que recebe e o
      // Salvar viraria um cadastro — um Endereço duplicado em vez do 404.
      setEditando((atual) => (atual && !lista.some((e) => e.id === atual) ? null : atual));
    } catch {
      setErroDaLista("Não foi possível falar com o servidor.");
    }
  }, [semSessao]);

  useEffect(() => {
    carregar();
  }, [carregar]);

  function abrir(endereco: Endereco | null) {
    setEditando(endereco ? endereco.id : "");
    setAberturas((n) => n + 1);
  }

  async function salvo() {
    setEditando(null);
    await carregar();
  }

  async function remover() {
    if (!aRemover) return;
    setEnviando(true);
    try {
      const resposta = await fetch(`/api/v1/enderecos/${encodeURIComponent(aRemover.id)}`, {
        method: "DELETE",
        credentials: "same-origin",
      });
      if (resposta.status === 401) {
        semSessao();
        return;
      }
      // O Dialog fecha em todo desfecho, e antes da mensagem: ele cobriria
      // justamente o Alert em que a falha é escrita.
      setARemover(null);
      if (!resposta.ok) {
        const corpo = await resposta.json().catch(() => null);
        setErroDaLista(corpo?.erro?.mensagem ?? "Não foi possível remover o Endereço.");
        // Sair antes do carregar(): é ele que limpa o erroDaLista no sucesso, e
        // a mensagem do envelope sumiria antes de ser lida.
        return;
      }
      await carregar();
    } catch {
      setARemover(null);
      setErroDaLista("Não foi possível falar com o servidor.");
    } finally {
      setEnviando(false);
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-medium">Meus endereços</h1>

      {erroDaLista && (
        <Alert variant="destructive">
          <AlertDescription>{erroDaLista}</AlertDescription>
        </Alert>
      )}

      {/* Antes da primeira leitura não há lista nem vazio: dizer "Nenhum
          Endereço cadastrado." enquanto a resposta está a caminho seria uma
          afirmação falsa sobre a conta. */}
      {enderecos?.length === 0 && editando === null && (
        <Card>
          <CardContent className="space-y-4 text-center">
            <p>Nenhum Endereço cadastrado.</p>
            {/* Ação única no estado vazio: não há nada a listar, ordenar ou
                filtrar, e um segundo botão só dividiria a atenção. */}
            <Button onClick={() => abrir(null)}>Cadastrar Endereço</Button>
          </CardContent>
        </Card>
      )}

      {enderecos?.map((endereco) => (
        <Card key={endereco.id}>
          <CardContent className="flex items-start justify-between gap-4">
            <EnderecoPorExtenso endereco={endereco} />
            <div className="flex shrink-0 gap-2">
              <Button variant="outline" size="sm" onClick={() => abrir(endereco)}>
                Editar
              </Button>
              <Button variant="outline" size="sm" onClick={() => setARemover(endereco)}>
                Remover
              </Button>
            </div>
          </CardContent>
        </Card>
      ))}

      {enderecos !== null && enderecos.length > 0 && editando === null && (
        <Button onClick={() => abrir(null)}>Cadastrar Endereço</Button>
      )}

      {editando !== null && (
        <Card>
          <CardHeader>
            <CardTitle>
              <h2>{editando === "" ? "Novo Endereço" : "Editar Endereço"}</h2>
            </CardTitle>
          </CardHeader>
          <CardContent>
            {/* A `key` refaz o formulário a cada Endereço aberto: o estado
                dele nasce do Endereço que recebe, uma vez. */}
            <FormularioDeEndereco
              key={`${editando}|${aberturas}`}
              endereco={enderecos?.find((e) => e.id === editando) ?? null}
              aoSalvar={salvo}
              aoCancelar={() => setEditando(null)}
            />
          </CardContent>
        </Card>
      )}

      {/* Remover é DELETE de verdade e não tem desfazer: por isso a confirmação
          nomeia o Endereço, em vez de perguntar "tem certeza?" sobre nada. */}
      <Dialog open={aRemover !== null} onOpenChange={(aberto) => !aberto && setARemover(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remover Endereço</DialogTitle>
            <DialogDescription>
              {aRemover &&
                `${aRemover.logradouro}, ${aRemover.numero} — ${aRemover.cidade}/${aRemover.uf}. Esta ação não pode ser desfeita. Seus Pedidos não mudam.`}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setARemover(null)} disabled={enviando}>
              Cancelar
            </Button>
            <Button onClick={remover} disabled={enviando}>
              {enviando ? "Removendo…" : "Remover"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
