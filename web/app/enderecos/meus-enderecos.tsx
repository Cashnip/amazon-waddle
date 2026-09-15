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
import { paraLogin } from "@/lib/destino";

// Nenhuma regra mora aqui (AD-10): a defesa é o Go, e quem decide é sempre o
// 400 dele — formato do CEP, as 27 siglas da UF, os obrigatórios e os tetos. O
// `required` não chega a barrar envio nenhum (o `noValidate` desliga a
// validação nativa), e fica pela semântica.
//
// O contrato do erro em linha é `dados.campo` (UX-DR16): o envelope diz de qual
// campo a mensagem é, e a tela liga os dois por `aria-describedby` e põe o foco
// no campo (UX-DR20c). O 409 do teto por Comprador não nomeia campo nenhum —
// não é um campo que está errado, é a conta que está cheia — e por isso sai no
// alerta do formulário, com a mensagem do servidor.

const CAMPOS = [
  "destinatario",
  "cep",
  "logradouro",
  "numero",
  "complemento",
  "bairro",
  "cidade",
  "uf",
] as const;
type Campo = (typeof CAMPOS)[number];

type Endereco = { id: string } & Record<Campo, string>;
type ErroDeCampo = { campo: Campo | null; mensagem: string };

const VAZIO: Record<Campo, string> = {
  destinatario: "",
  cep: "",
  logradouro: "",
  numero: "",
  complemento: "",
  bairro: "",
  cidade: "",
  uf: "",
};

// O CEP é guardado em oito dígitos; a máscara é da tela, e só na exibição.
function comMascara(cep: string): string {
  return cep.length === 8 ? `${cep.slice(0, 5)}-${cep.slice(5)}` : cep;
}

export function MeusEnderecos() {
  const router = useRouter();
  const [enderecos, setEnderecos] = useState<Endereco[] | null>(null);
  const [erroDaLista, setErroDaLista] = useState<string | null>(null);
  // `null` = nenhum formulário aberto; "" = cadastrando; um id = editando.
  const [editando, setEditando] = useState<string | null>(null);
  const [valores, setValores] = useState<Record<Campo, string>>(VAZIO);
  const [erro, setErro] = useState<ErroDeCampo | null>(null);
  const [enviando, setEnviando] = useState(false);
  const [aRemover, setARemover] = useState<Endereco | null>(null);

  const refs: Record<Campo, React.RefObject<HTMLInputElement | null>> = {
    destinatario: useRef<HTMLInputElement>(null),
    cep: useRef<HTMLInputElement>(null),
    logradouro: useRef<HTMLInputElement>(null),
    numero: useRef<HTMLInputElement>(null),
    complemento: useRef<HTMLInputElement>(null),
    bairro: useRef<HTMLInputElement>(null),
    cidade: useRef<HTMLInputElement>(null),
    uf: useRef<HTMLInputElement>(null),
  };

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
      setEnderecos(corpo ?? []);
    } catch {
      setErroDaLista("Não foi possível falar com o servidor.");
    }
  }, [semSessao]);

  useEffect(() => {
    carregar();
  }, [carregar]);

  function abrir(endereco: Endereco | null) {
    setErro(null);
    setEditando(endereco ? endereco.id : "");
    setValores(
      endereco
        ? {
            destinatario: endereco.destinatario,
            // Mascarado para quem edita ler o CEP como o escreveu; o Go aceita
            // as duas formas.
            cep: comMascara(endereco.cep),
            logradouro: endereco.logradouro,
            numero: endereco.numero,
            complemento: endereco.complemento,
            bairro: endereco.bairro,
            cidade: endereco.cidade,
            uf: endereco.uf,
          }
        : VAZIO,
    );
  }

  function recusar(erroDeCampo: ErroDeCampo) {
    setErro(erroDeCampo);
    if (erroDeCampo.campo) refs[erroDeCampo.campo].current?.focus();
  }

  async function salvar(evento: React.FormEvent) {
    evento.preventDefault();
    setEnviando(true);
    setErro(null);
    // Cadastrar e editar são a mesma etiqueta inteira: muda o método e o
    // endereço da rota, e mais nada.
    const editandoID = editando !== "" ? editando : null;
    try {
      const resposta = await fetch(
        editandoID ? `/api/v1/enderecos/${encodeURIComponent(editandoID)}` : "/api/v1/enderecos",
        {
          method: editandoID ? "PUT" : "POST",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(valores),
        },
      );
      if (resposta.status === 401) {
        semSessao();
        return;
      }
      const corpo = await resposta.json().catch(() => null);
      if (!resposta.ok) {
        // A mensagem exibida é sempre a do envelope do servidor (AD-14): é ela
        // que fala a Voice and Tone e que nomeia o limiar, que mora na Config e
        // não no JavaScript. O campo é filtrado contra CAMPOS — um nome que a
        // tela não conhece cai no alerta em vez de sumir da tela inteira.
        recusar({
          campo: CAMPOS.find((c) => c === corpo?.erro?.dados?.campo) ?? null,
          mensagem: corpo?.erro?.mensagem ?? "Não foi possível salvar o Endereço.",
        });
        return;
      }
      setEditando(null);
      await carregar();
    } catch {
      recusar({ campo: null, mensagem: "Não foi possível falar com o servidor." });
    } finally {
      // No finally, e não no fim: o `return` do 401 pula o corpo da função, e
      // sem isto os botões ficariam desabilitados esperando uma navegação que
      // pode nem trocar a rota.
      setEnviando(false);
    }
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

  function campoDe(campo: Campo, rotulo: string, autoComplete: string, dica?: string) {
    const comErro = erro?.campo === campo;
    const idErro = `${campo}-erro`;
    const idDica = `${campo}-dica`;
    const descrito = [dica ? idDica : null, comErro ? idErro : null].filter(Boolean).join(" ");
    return (
      <div className="space-y-2">
        <Label htmlFor={campo}>{rotulo}</Label>
        <Input
          id={campo}
          name={campo}
          autoComplete={autoComplete}
          required={campo !== "complemento"}
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
          </p>
        )}
      </div>
    );
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
            <address className="space-y-1 not-italic">
              <p className="font-medium">{endereco.destinatario}</p>
              <p>
                {endereco.logradouro}, {endereco.numero}
                {endereco.complemento && ` — ${endereco.complemento}`}
              </p>
              <p>
                {endereco.bairro} — {endereco.cidade}/{endereco.uf}
              </p>
              <p className="text-muted-foreground">CEP {comMascara(endereco.cep)}</p>
            </address>
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
            {/* noValidate: a mensagem que vale é a do Go, e a bolha nativa do
                navegador a esconderia antes de a requisição sair. */}
            <form className="space-y-4" onSubmit={salvar} noValidate>
              {campoDe("destinatario", "Quem recebe", "name")}
              {campoDe("cep", "CEP", "postal-code", "Oito dígitos, com ou sem hífen.")}
              {campoDe("logradouro", "Rua ou avenida", "address-line1")}
              {campoDe("numero", "Número", "address-line2")}
              {campoDe("complemento", "Complemento (opcional)", "address-line3")}
              {campoDe("bairro", "Bairro", "address-level3")}
              {campoDe("cidade", "Cidade", "address-level2")}
              {campoDe("uf", "UF", "address-level1", "Duas letras, como SP.")}

              {/* Erro sem campo — rede fora, ou o 409 do teto por Comprador,
                  que não é de campo nenhum. */}
              {erro && !erro.campo && (
                <Alert variant="destructive">
                  <AlertDescription>{erro.mensagem}</AlertDescription>
                </Alert>
              )}

              <div className="flex gap-2">
                <Button type="submit" disabled={enviando}>
                  {enviando ? "Salvando…" : "Salvar Endereço"}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  disabled={enviando}
                  onClick={() => setEditando(null)}
                >
                  Cancelar
                </Button>
              </div>
            </form>
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
