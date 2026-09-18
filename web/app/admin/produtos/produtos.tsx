"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
} from "@/components/ui/pagination";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ImagemDoProduto } from "@/components/imagem-do-produto";
import { FALHA_DE_REDE, pedir } from "@/lib/pedir";
import { centavosParaReais, formatarPreco, reaisParaCentavos } from "@/lib/preco";

// A tela Produtos (3.3), no molde de Vendedores. Toda regra é do Go: tetos,
// preço maior que zero, imagem da lista, Vendedor e Categoria existentes. A
// única conta daqui é o preço em reais virar centavos, por texto (AD-3).
//
// A página vive na URL (`?pagina=`): recarregar e voltar reproduzem a lista.
// Produto não se remove (FR-9): desativa-se, e o inativo continua na lista.

type Referencia = { id: string; nome: string };
type Produto = {
  id: string;
  nome: string;
  descricao: string;
  preco_centavos: number;
  imagem_url: string;
  estoque_total: number;
  ativo: boolean;
  vendedor: Referencia;
  categoria: Referencia;
};
type Listagem = { itens: Produto[]; pagina: number; por_pagina: number; total: number };
type Vendedor = Referencia & { ativo: boolean };

const CAMPOS = [
  "nome",
  "descricao",
  "preco_centavos",
  "estoque_total",
  "vendedor_id",
  "categoria_id",
  "imagem_url",
] as const;
type Campo = (typeof CAMPOS)[number];
type ErroDoForm = { campo: Campo | null; mensagem: string };

// O Select do Radix não aceita valor vazio: "Sem imagem" tem um nome próprio.
const SEM_IMAGEM = "sem-imagem";

const FORM_VAZIO = { nome: "", descricao: "", preco: "", estoque: "", vendedor: "", categoria: "", imagem: SEM_IMAGEM };

function rotaDe(id: string) {
  return `/api/v1/admin/produtos/${encodeURIComponent(id)}`;
}

function corpoDe(p: Produto, ativo: boolean) {
  return {
    nome: p.nome,
    descricao: p.descricao,
    preco_centavos: p.preco_centavos,
    imagem_url: p.imagem_url,
    vendedor_id: p.vendedor.id,
    categoria_id: p.categoria.id,
    ativo,
  };
}

export function Produtos() {
  const router = useRouter();
  const pagina = Math.max(1, Number.parseInt(useSearchParams().get("pagina") ?? "1", 10) || 1);
  const [lista, setLista] = useState<Listagem | null>(null);
  const [erroDaLista, setErroDaLista] = useState<string | null>(null);
  const [vendedores, setVendedores] = useState<Vendedor[]>([]);
  const [categorias, setCategorias] = useState<Referencia[]>([]);
  const [midias, setMidias] = useState<string[]>([]);
  // `null` = nenhum formulário; "" = cadastrando; um id = editando.
  const [editando, setEditando] = useState<string | null>(null);
  const [form, setForm] = useState(FORM_VAZIO);
  const [erro, setErro] = useState<ErroDoForm | null>(null);
  const [enviando, setEnviando] = useState(false);
  const refs = useRef<Partial<Record<Campo, HTMLElement | null>>>({});

  const carregar = useCallback(async () => {
    try {
      const { resposta, json } = await pedir(`/api/v1/admin/produtos?pagina=${pagina}`, "GET");
      if (resposta.status === 404) {
        router.replace("/admin/entrar");
        return;
      }
      if (!resposta.ok) {
        setErroDaLista(json?.erro?.mensagem ?? "Não foi possível ler os Produtos.");
        return;
      }
      setErroDaLista(null);
      setLista(json);
    } catch {
      setErroDaLista(FALHA_DE_REDE);
    }
  }, [pagina, router]);

  useEffect(() => {
    carregar();
  }, [carregar]);

  // As opções dos três Selects. Falha aqui não tira a lista da tela: o
  // formulário só fica sem opções, e o Go recusa o que vier vazio.
  useEffect(() => {
    function ler<T>(url: string, definir: (valor: T) => void) {
      pedir(url, "GET")
        .then(({ resposta, json }) => resposta.ok && definir(json))
        .catch(() => {});
    }
    ler("/api/v1/admin/vendedores", setVendedores);
    ler("/api/v1/admin/categorias", setCategorias);
    ler("/api/v1/admin/midias", setMidias);
  }, []);

  async function falhou(mensagem: string) {
    await carregar();
    setErroDaLista(mensagem);
  }

  function abrir(produto: Produto | null) {
    setErro(null);
    setEditando(produto ? produto.id : "");
    setForm(
      produto
        ? {
            nome: produto.nome,
            descricao: produto.descricao,
            preco: centavosParaReais(produto.preco_centavos),
            estoque: String(produto.estoque_total),
            vendedor: produto.vendedor.id,
            categoria: produto.categoria.id,
            imagem: produto.imagem_url || SEM_IMAGEM,
          }
        : FORM_VAZIO,
    );
  }

  function mostrarErro(campo: Campo | null, mensagem: string) {
    setErro({ campo, mensagem });
    if (campo) refs.current[campo]?.focus();
  }

  async function salvar(evento: React.FormEvent) {
    evento.preventDefault();
    if (editando === null) return;
    const atual = lista?.itens.find((p) => p.id === editando);
    // Editando um Produto que saiu da página: sem isto cairia no POST.
    if (editando !== "" && !atual) {
      setEditando(null);
      await falhou("Produto não encontrado.");
      return;
    }
    const preco = reaisParaCentavos(form.preco);
    if (preco === null) {
      mostrarErro("preco_centavos", "Informe o preço em reais, como 12,90.");
      return;
    }
    const corpo = {
      nome: form.nome,
      descricao: form.descricao,
      preco_centavos: preco,
      imagem_url: form.imagem === SEM_IMAGEM ? "" : form.imagem,
      vendedor_id: form.vendedor,
      categoria_id: form.categoria,
    };
    // O Estoque total só existe na criação: o ajuste é da 3.4. Texto que não
    // é inteiro vai como -1, e a mensagem que volta é a do Go, com o teto.
    // isSafeInteger: dígitos demais não podem virar um número que o Go nem
    // decodifica — o erro tem de cair no campo, e não num 400 genérico.
    const digitado = form.estoque.trim();
    const estoque = /^\d+$/.test(digitado) && Number.isSafeInteger(Number(digitado)) ? Number(digitado) : -1;
    setEnviando(true);
    setErro(null);
    try {
      const { resposta, json } = atual
        ? await pedir(rotaDe(atual.id), "PUT", { ...corpo, ativo: atual.ativo })
        : await pedir("/api/v1/admin/produtos", "POST", { ...corpo, estoque_total: estoque });
      if (resposta.status === 404) {
        setEditando(null);
        await falhou(json?.erro?.mensagem ?? "Produto não encontrado.");
        return;
      }
      if (!resposta.ok) {
        const campo = CAMPOS.find((c) => c === json?.erro?.dados?.campo) ?? null;
        mostrarErro(campo, json?.erro?.mensagem ?? "Não foi possível salvar o Produto.");
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
  async function definirAtivo(produto: Produto, ativo: boolean) {
    setEnviando(true);
    try {
      const { resposta, json } = await pedir(rotaDe(produto.id), "PUT", corpoDe(produto, ativo));
      if (!resposta.ok) {
        await falhou(json?.erro?.mensagem ?? "Não foi possível salvar o Produto.");
        return;
      }
      await carregar();
    } catch {
      setErroDaLista(FALHA_DE_REDE);
    } finally {
      setEnviando(false);
    }
  }

  const paginas = lista ? Math.max(1, Math.ceil(lista.total / lista.por_pagina)) : 1;

  // Liga cada campo à sua mensagem em linha e ao foco.
  function ligar(campo: Campo) {
    const comErro = erro?.campo === campo;
    return {
      id: campo,
      ref: (el: HTMLElement | null) => {
        refs.current[campo] = el;
      },
      "aria-invalid": comErro || undefined,
      "aria-describedby": comErro ? `${campo}-erro` : undefined,
    };
  }
  function mensagem(campo: Campo) {
    return (
      erro?.campo === campo && (
        <p id={`${campo}-erro`} className="text-destructive text-sm" role="alert">
          {erro.mensagem}
        </p>
      )
    );
  }

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-medium">Produtos</h1>

      {erroDaLista && (
        <Alert variant="destructive">
          <AlertDescription>{erroDaLista}</AlertDescription>
        </Alert>
      )}

      {lista?.total === 0 && editando === null && (
        <Card>
          <CardContent className="space-y-4 text-center">
            <p>Nenhum Produto cadastrado.</p>
            <Button onClick={() => abrir(null)}>Cadastrar Produto</Button>
          </CardContent>
        </Card>
      )}

      {lista !== null && lista.total > 0 && (
        <>
          {editando === null && <Button onClick={() => abrir(null)}>Cadastrar Produto</Button>}
          {/* A tabela rola dentro do próprio contêiner: a página nunca. */}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-20">
                  <span className="sr-only">Imagem</span>
                </TableHead>
                <TableHead>Nome</TableHead>
                <TableHead className="text-right">Preço</TableHead>
                <TableHead className="text-right">Estoque total</TableHead>
                <TableHead>Vendedor</TableHead>
                <TableHead>Categoria</TableHead>
                <TableHead>Situação</TableHead>
                <TableHead className="text-right">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {lista.itens.map((p) => (
                <TableRow key={p.id}>
                  <TableCell>
                    <ImagemDoProduto src={p.imagem_url} nome={p.nome} compacta className="w-16" />
                  </TableCell>
                  <TableCell className="min-w-48 font-medium whitespace-normal">{p.nome}</TableCell>
                  <TableCell className="text-right tabular-nums">{formatarPreco(p.preco_centavos)}</TableCell>
                  <TableCell className="text-right tabular-nums">{p.estoque_total.toLocaleString("pt-BR")}</TableCell>
                  <TableCell>{p.vendedor.nome}</TableCell>
                  <TableCell>{p.categoria.nome}</TableCell>
                  <TableCell>{p.ativo ? "Ativo" : "Inativo"}</TableCell>
                  <TableCell>
                    <div className="flex justify-end gap-2">
                      <Button variant="outline" size="sm" disabled={enviando} onClick={() => abrir(p)}>
                        Editar
                      </Button>
                      <Button variant="outline" size="sm" disabled={enviando} onClick={() => definirAtivo(p, !p.ativo)}>
                        {p.ativo ? "Desativar" : "Reativar"}
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {lista.itens.length === 0 && <p>Esta página não tem Produtos.</p>}
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-muted-foreground text-sm">
              Página {lista.pagina} de {paginas} · {lista.total.toLocaleString("pt-BR")}{" "}
              {lista.total === 1 ? "Produto" : "Produtos"}
            </p>
            {paginas > 1 && (
              <Pagination aria-label="Páginas de Produtos" className="mx-0 w-auto">
                <PaginationContent>
                  {pagina > 1 && (
                    <PaginationItem>
                      <PaginationLink size="default" href={`?pagina=${pagina - 1}`}>
                        Anterior
                      </PaginationLink>
                    </PaginationItem>
                  )}
                  {pagina < paginas && (
                    <PaginationItem>
                      <PaginationLink size="default" href={`?pagina=${pagina + 1}`}>
                        Próxima
                      </PaginationLink>
                    </PaginationItem>
                  )}
                </PaginationContent>
              </Pagination>
            )}
          </div>
        </>
      )}

      {editando !== null && (
        <Card>
          <CardHeader>
            <CardTitle>
              <h2>{editando === "" ? "Novo Produto" : "Editar Produto"}</h2>
            </CardTitle>
          </CardHeader>
          <CardContent>
            {/* noValidate: a mensagem que vale é a do Go, com o limite. */}
            <form className="space-y-4" onSubmit={salvar} noValidate>
              <div className="space-y-2">
                <Label htmlFor="nome">Nome</Label>
                <Input
                  {...ligar("nome")}
                  name="nome"
                  autoFocus
                  value={form.nome}
                  onChange={(e) => setForm({ ...form, nome: e.target.value })}
                />
                {mensagem("nome")}
              </div>

              <div className="space-y-2">
                <Label htmlFor="descricao">Descrição</Label>
                <textarea
                  {...ligar("descricao")}
                  name="descricao"
                  rows={5}
                  className="border-input focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:border-destructive w-full rounded-lg border bg-transparent px-2.5 py-2 text-sm outline-none focus-visible:ring-3"
                  value={form.descricao}
                  onChange={(e) => setForm({ ...form, descricao: e.target.value })}
                />
                {mensagem("descricao")}
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="preco_centavos">Preço (R$)</Label>
                  <Input
                    {...ligar("preco_centavos")}
                    name="preco"
                    inputMode="decimal"
                    placeholder="12,90"
                    value={form.preco}
                    onChange={(e) => setForm({ ...form, preco: e.target.value })}
                  />
                  {mensagem("preco_centavos")}
                </div>
                {editando === "" && (
                  <div className="space-y-2">
                    <Label htmlFor="estoque_total">Estoque total</Label>
                    <Input
                      {...ligar("estoque_total")}
                      name="estoque"
                      inputMode="numeric"
                      value={form.estoque}
                      onChange={(e) => setForm({ ...form, estoque: e.target.value })}
                    />
                    {mensagem("estoque_total")}
                  </div>
                )}
              </div>

              <div className="grid gap-4 sm:grid-cols-3">
                <div className="space-y-2">
                  <Label htmlFor="vendedor_id">Vendedor</Label>
                  <Select value={form.vendedor} onValueChange={(v) => setForm({ ...form, vendedor: v })}>
                    <SelectTrigger {...ligar("vendedor_id")} className="w-full">
                      <SelectValue placeholder="Escolha o Vendedor" />
                    </SelectTrigger>
                    <SelectContent>
                      {vendedores.map((v) => (
                        <SelectItem key={v.id} value={v.id}>
                          {v.ativo ? v.nome : `${v.nome} (Inativo)`}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {mensagem("vendedor_id")}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="categoria_id">Categoria</Label>
                  <Select value={form.categoria} onValueChange={(v) => setForm({ ...form, categoria: v })}>
                    <SelectTrigger {...ligar("categoria_id")} className="w-full">
                      <SelectValue placeholder="Escolha a Categoria" />
                    </SelectTrigger>
                    <SelectContent>
                      {categorias.map((c) => (
                        <SelectItem key={c.id} value={c.id}>
                          {c.nome}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {mensagem("categoria_id")}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="imagem_url">Imagem</Label>
                  <Select value={form.imagem} onValueChange={(v) => setForm({ ...form, imagem: v })}>
                    <SelectTrigger {...ligar("imagem_url")} className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={SEM_IMAGEM}>Sem imagem</SelectItem>
                      {midias.map((m) => (
                        <SelectItem key={m} value={m}>
                          {m.slice(m.lastIndexOf("/") + 1)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {mensagem("imagem_url")}
                </div>
              </div>

              <ImagemDoProduto
                src={form.imagem === SEM_IMAGEM ? "" : form.imagem}
                nome={form.nome || "Sem imagem"}
                className="max-w-48"
              />

              {erro && !erro.campo && (
                <Alert variant="destructive">
                  <AlertDescription>{erro.mensagem}</AlertDescription>
                </Alert>
              )}

              <div className="flex gap-2">
                <Button type="submit" disabled={enviando}>
                  {enviando ? "Salvando…" : "Salvar Produto"}
                </Button>
                <Button type="button" variant="outline" disabled={enviando} onClick={() => setEditando(null)}>
                  Cancelar
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
