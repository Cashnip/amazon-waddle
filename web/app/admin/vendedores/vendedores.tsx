"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertAction, AlertDescription } from "@/components/ui/alert";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

// Nenhuma regra mora aqui (AD-10): o teto do nome, o duplicado e a recusa de
// remover Vendedor com Produtos são todos do Go, e a mensagem exibida é sempre
// a do envelope. O erro em linha é ligado ao campo por `dados.campo`.
//
// 404 na leitura da lista é a guarda do prefixo: a Sessão de Administrador
// sumiu, e a tela volta ao login administrativo. 404 numa mutação pode ser só
// o Vendedor que não existe mais — a lista é relida, e é ela que decide.

type Vendedor = { id: string; nome: string; ativo: boolean };
type ErroDoForm = { campo: "nome" | null; mensagem: string };

const FALHA_DE_REDE = "Não foi possível falar com o servidor.";

async function pedir(url: string, method: string, corpo?: unknown) {
  const resposta = await fetch(url, {
    method,
    credentials: "same-origin",
    headers: corpo === undefined ? undefined : { "Content-Type": "application/json" },
    body: corpo === undefined ? undefined : JSON.stringify(corpo),
  });
  const json = resposta.status === 204 ? null : await resposta.json().catch(() => null);
  return { resposta, json };
}

function rotaDe(id: string) {
  return `/api/v1/admin/vendedores/${encodeURIComponent(id)}`;
}

export function Vendedores() {
  const router = useRouter();
  const [vendedores, setVendedores] = useState<Vendedor[] | null>(null);
  const [erroDaLista, setErroDaLista] = useState<string | null>(null);
  // A recusa de remover Vendedor com Produtos: o Alert oferece desativar.
  const [recusa, setRecusa] = useState<{ vendedor: Vendedor; mensagem: string } | null>(null);
  // `null` = nenhum formulário; "" = cadastrando; um id = editando.
  const [editando, setEditando] = useState<string | null>(null);
  const [nome, setNome] = useState("");
  const [erro, setErro] = useState<ErroDoForm | null>(null);
  const [enviando, setEnviando] = useState(false);
  const [aRemover, setARemover] = useState<Vendedor | null>(null);
  const campoNome = useRef<HTMLInputElement>(null);

  const carregar = useCallback(async () => {
    try {
      const { resposta, json } = await pedir("/api/v1/admin/vendedores", "GET");
      if (resposta.status === 404) {
        router.replace("/admin/entrar");
        return;
      }
      if (!resposta.ok) {
        setErroDaLista(json?.erro?.mensagem ?? "Não foi possível ler os Vendedores.");
        return;
      }
      setErroDaLista(null);
      setVendedores(json ?? []);
    } catch {
      setErroDaLista(FALHA_DE_REDE);
    }
  }, [router]);

  useEffect(() => {
    carregar();
  }, [carregar]);

  // Falha de mutação fora do formulário: relê a lista (que manda ao login se a
  // Sessão caiu) e só então escreve a mensagem — o carregar() a limparia.
  async function falhou(mensagem: string) {
    await carregar();
    setErroDaLista(mensagem);
  }

  function abrir(vendedor: Vendedor | null) {
    setErro(null);
    setRecusa(null);
    setEditando(vendedor ? vendedor.id : "");
    setNome(vendedor ? vendedor.nome : "");
  }

  async function salvar(evento: React.FormEvent) {
    evento.preventDefault();
    if (editando === null) return;
    setEnviando(true);
    setErro(null);
    const atual = vendedores?.find((v) => v.id === editando);
    // Editando um Vendedor que saiu da lista (removido no meio da edição): sem
    // isto o formulário cairia no POST e criaria um Vendedor novo.
    if (editando !== "" && !atual) {
      setEditando(null);
      setEnviando(false);
      await falhou("Vendedor não encontrado.");
      return;
    }
    try {
      const { resposta, json } = atual
        ? await pedir(rotaDe(atual.id), "PUT", { nome, ativo: atual.ativo })
        : await pedir("/api/v1/admin/vendedores", "POST", { nome });
      if (resposta.status === 404) {
        setEditando(null);
        await falhou(json?.erro?.mensagem ?? "Vendedor não encontrado.");
        return;
      }
      if (!resposta.ok) {
        const campo = json?.erro?.dados?.campo === "nome" ? "nome" : null;
        setErro({ campo, mensagem: json?.erro?.mensagem ?? "Não foi possível salvar o Vendedor." });
        if (campo) campoNome.current?.focus();
        return;
      }
      setEditando(null);
      await carregar();
    } catch {
      setErro({ campo: null, mensagem: FALHA_DE_REDE });
    } finally {
      setEnviando(false);
    }
  }

  // Desativar e reativar são o mesmo PUT da linha inteira, com `ativo` trocado.
  async function definirAtivo(vendedor: Vendedor, ativo: boolean) {
    setEnviando(true);
    setRecusa(null);
    try {
      const { resposta, json } = await pedir(rotaDe(vendedor.id), "PUT", { nome: vendedor.nome, ativo });
      if (!resposta.ok) {
        await falhou(json?.erro?.mensagem ?? "Não foi possível salvar o Vendedor.");
        return;
      }
      await carregar();
    } catch {
      setErroDaLista(FALHA_DE_REDE);
    } finally {
      setEnviando(false);
    }
  }

  async function remover() {
    if (!aRemover) return;
    const vendedor = aRemover;
    setEnviando(true);
    try {
      const { resposta, json } = await pedir(rotaDe(vendedor.id), "DELETE");
      // O Dialog fecha antes da mensagem: ele cobriria o Alert.
      setARemover(null);
      if (resposta.status === 409) {
        setRecusa({ vendedor, mensagem: json?.erro?.mensagem ?? "Este Vendedor não pode ser removido." });
        return;
      }
      if (!resposta.ok) {
        await falhou(json?.erro?.mensagem ?? "Não foi possível remover o Vendedor.");
        return;
      }
      await carregar();
    } catch {
      setARemover(null);
      setErroDaLista(FALHA_DE_REDE);
    } finally {
      setEnviando(false);
    }
  }

  const comErro = erro?.campo === "nome";

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-medium">Vendedores</h1>

      {erroDaLista && (
        <Alert variant="destructive">
          <AlertDescription>{erroDaLista}</AlertDescription>
        </Alert>
      )}

      {recusa && (
        <Alert variant="destructive">
          <AlertDescription>{recusa.mensagem}</AlertDescription>
          {recusa.vendedor.ativo && (
            <AlertAction>
              <Button size="sm" disabled={enviando} onClick={() => definirAtivo(recusa.vendedor, false)}>
                Desativar
              </Button>
            </AlertAction>
          )}
        </Alert>
      )}

      {/* Antes da primeira leitura não há lista nem vazio. */}
      {vendedores?.length === 0 && editando === null && (
        <Card>
          <CardContent className="space-y-4 text-center">
            <p>Nenhum Vendedor cadastrado.</p>
            <Button onClick={() => abrir(null)}>Cadastrar Vendedor</Button>
          </CardContent>
        </Card>
      )}

      {vendedores !== null && vendedores.length > 0 && (
        <>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nome</TableHead>
                <TableHead>Situação</TableHead>
                <TableHead className="text-right">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {vendedores.map((v) => (
                <TableRow key={v.id}>
                  <TableCell className="font-medium whitespace-normal">{v.nome}</TableCell>
                  <TableCell>{v.ativo ? "Ativo" : "Inativo"}</TableCell>
                  <TableCell>
                    <div className="flex flex-wrap justify-end gap-2">
                      <Button variant="outline" size="sm" disabled={enviando} onClick={() => abrir(v)}>
                        Editar
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={enviando}
                        onClick={() => definirAtivo(v, !v.ativo)}
                      >
                        {v.ativo ? "Desativar" : "Reativar"}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={enviando}
                        onClick={() => {
                          setRecusa(null);
                          setARemover(v);
                        }}
                      >
                        Remover
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {editando === null && <Button onClick={() => abrir(null)}>Cadastrar Vendedor</Button>}
        </>
      )}

      {editando !== null && (
        <Card>
          <CardHeader>
            <CardTitle>
              <h2>{editando === "" ? "Novo Vendedor" : "Editar Vendedor"}</h2>
            </CardTitle>
          </CardHeader>
          <CardContent>
            {/* noValidate: a mensagem que vale é a do Go, com o limite. */}
            <form className="space-y-4" onSubmit={salvar} noValidate>
              <div className="space-y-2">
                <Label htmlFor="nome">Nome</Label>
                <Input
                  id="nome"
                  name="nome"
                  required
                  autoFocus
                  ref={campoNome}
                  aria-invalid={comErro || undefined}
                  aria-describedby={comErro ? "nome-erro" : undefined}
                  value={nome}
                  onChange={(e) => setNome(e.target.value)}
                />
                {comErro && (
                  <p id="nome-erro" className="text-destructive text-sm" role="alert">
                    {erro.mensagem}
                  </p>
                )}
              </div>

              {erro && !erro.campo && (
                <Alert variant="destructive">
                  <AlertDescription>{erro.mensagem}</AlertDescription>
                </Alert>
              )}

              <div className="flex gap-2">
                <Button type="submit" disabled={enviando}>
                  {enviando ? "Salvando…" : "Salvar Vendedor"}
                </Button>
                <Button type="button" variant="outline" disabled={enviando} onClick={() => setEditando(null)}>
                  Cancelar
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <Dialog open={aRemover !== null} onOpenChange={(aberto) => !aberto && setARemover(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remover Vendedor</DialogTitle>
            <DialogDescription>
              {aRemover && `${aRemover.nome}. Esta ação não pode ser desfeita.`}
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
