"use client";

import { useCallback, useEffect, useRef, useState } from "react";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { FALHA_DE_REDE, pedir } from "@/lib/pedir";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

// O molde de Vendedores (3.1), sem situação: a Categoria não se desativa. O
// teto do nome, o duplicado e a recusa de remover Categoria com Produtos são
// do Go, e a recusa exibe a mensagem do servidor, que já traz a contagem.

type Categoria = { id: string; nome: string };
type ErroDoForm = { campo: "nome" | null; mensagem: string };

function rotaDe(id: string) {
  return `/api/v1/admin/categorias/${encodeURIComponent(id)}`;
}

export function Categorias() {
  const router = useRouter();
  const [categorias, setCategorias] = useState<Categoria[] | null>(null);
  const [erroDaLista, setErroDaLista] = useState<string | null>(null);
  const [recusa, setRecusa] = useState<string | null>(null);
  // `null` = nenhum formulário; "" = cadastrando; um id = editando.
  const [editando, setEditando] = useState<string | null>(null);
  const [nome, setNome] = useState("");
  const [erro, setErro] = useState<ErroDoForm | null>(null);
  const [enviando, setEnviando] = useState(false);
  const [aRemover, setARemover] = useState<Categoria | null>(null);
  const campoNome = useRef<HTMLInputElement>(null);

  const carregar = useCallback(async () => {
    try {
      const { resposta, json } = await pedir("/api/v1/admin/categorias", "GET");
      if (resposta.status === 404) {
        router.replace("/admin/entrar");
        return;
      }
      if (!resposta.ok) {
        setErroDaLista(json?.erro?.mensagem ?? "Não foi possível ler as Categorias.");
        return;
      }
      setErroDaLista(null);
      setCategorias(json ?? []);
    } catch {
      setErroDaLista(FALHA_DE_REDE);
    }
  }, [router]);

  useEffect(() => {
    carregar();
  }, [carregar]);

  async function falhou(mensagem: string) {
    await carregar();
    setErroDaLista(mensagem);
  }

  function abrir(categoria: Categoria | null) {
    setErro(null);
    setRecusa(null);
    setEditando(categoria ? categoria.id : "");
    setNome(categoria ? categoria.nome : "");
  }

  async function salvar(evento: React.FormEvent) {
    evento.preventDefault();
    if (editando === null) return;
    setEnviando(true);
    setErro(null);
    const atual = categorias?.find((c) => c.id === editando);
    // Editando uma Categoria que saiu da lista: sem isto o formulário cairia
    // no POST e criaria outra.
    if (editando !== "" && !atual) {
      setEditando(null);
      setEnviando(false);
      await falhou("Categoria não encontrada.");
      return;
    }
    try {
      const { resposta, json } = atual
        ? await pedir(rotaDe(atual.id), "PUT", { nome })
        : await pedir("/api/v1/admin/categorias", "POST", { nome });
      if (resposta.status === 404) {
        setEditando(null);
        await falhou(json?.erro?.mensagem ?? "Categoria não encontrada.");
        return;
      }
      if (!resposta.ok) {
        const campo = json?.erro?.dados?.campo === "nome" ? "nome" : null;
        setErro({ campo, mensagem: json?.erro?.mensagem ?? "Não foi possível salvar a Categoria." });
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

  async function remover() {
    if (!aRemover) return;
    const categoria = aRemover;
    setEnviando(true);
    try {
      const { resposta, json } = await pedir(rotaDe(categoria.id), "DELETE");
      // O Dialog fecha antes da mensagem: ele cobriria o Alert.
      setARemover(null);
      if (resposta.status === 409) {
        setRecusa(json?.erro?.mensagem ?? "Esta Categoria não pode ser removida.");
        return;
      }
      if (!resposta.ok) {
        await falhou(json?.erro?.mensagem ?? "Não foi possível remover a Categoria.");
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
      <h1 className="text-2xl font-medium">Categorias</h1>

      {erroDaLista && (
        <Alert variant="destructive">
          <AlertDescription>{erroDaLista}</AlertDescription>
        </Alert>
      )}

      {recusa && (
        <Alert variant="destructive">
          <AlertDescription>{recusa}</AlertDescription>
        </Alert>
      )}

      {categorias?.length === 0 && editando === null && (
        <Card>
          <CardContent className="space-y-4 text-center">
            <p>Nenhuma Categoria cadastrada.</p>
            <Button onClick={() => abrir(null)}>Cadastrar Categoria</Button>
          </CardContent>
        </Card>
      )}

      {categorias !== null && categorias.length > 0 && (
        <>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nome</TableHead>
                <TableHead className="text-right">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {categorias.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="font-medium whitespace-normal">{c.nome}</TableCell>
                  <TableCell>
                    <div className="flex flex-wrap justify-end gap-2">
                      <Button variant="outline" size="sm" disabled={enviando} onClick={() => abrir(c)}>
                        Renomear
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={enviando}
                        onClick={() => {
                          setRecusa(null);
                          setARemover(c);
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
          {editando === null && <Button onClick={() => abrir(null)}>Cadastrar Categoria</Button>}
        </>
      )}

      {editando !== null && (
        <Card>
          <CardHeader>
            <CardTitle>
              <h2>{editando === "" ? "Nova Categoria" : "Renomear Categoria"}</h2>
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
                  {enviando ? "Salvando…" : "Salvar Categoria"}
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
            <DialogTitle>Remover Categoria</DialogTitle>
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
