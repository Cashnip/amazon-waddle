"use client";

import { useId, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { paraLogin } from "@/lib/destino";
import {
  FALHA_AO_SALVAR,
  VAZIO,
  comMascara,
  erroDeSalvar,
  requisicaoDeSalvar,
  type Campo,
  type Endereco,
} from "@/lib/endereco";
import { FALHA_DE_REDE } from "@/lib/pedir";

// O formulário de Endereço, um só para "Meus endereços" e para o checkout (5.2).
//
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

type ErroDeCampo = { campo: Campo | null; mensagem: string };

function valoresDe(endereco: Endereco | null): Record<Campo, string> {
  if (!endereco) return VAZIO;
  return {
    destinatario: endereco.destinatario,
    // Mascarado para quem edita ler o CEP como o escreveu; o Go aceita as duas
    // formas.
    cep: comMascara(endereco.cep),
    logradouro: endereco.logradouro,
    numero: endereco.numero,
    complemento: endereco.complemento,
    bairro: endereco.bairro,
    cidade: endereco.cidade,
    uf: endereco.uf,
  };
}

// `endereco` nulo é cadastro; com valor, é edição. O estado nasce dele uma vez:
// quem troca de Endereço remonta o formulário com outra `key`.
export function FormularioDeEndereco({
  endereco,
  aoSalvar,
  aoCancelar,
}: {
  endereco: Endereco | null;
  // Recebe o Endereço como o Go o devolveu — no cadastro, é por ele que o
  // checkout sabe qual marcar.
  aoSalvar: (salvo: Endereco) => void | Promise<void>;
  // Sem ele, não há botão Cancelar: o Comprador sem Endereço no checkout não
  // tem para onde voltar dentro do passo.
  aoCancelar?: () => void;
}) {
  const router = useRouter();
  const prefixo = useId();
  const [valores, setValores] = useState<Record<Campo, string>>(() => valoresDe(endereco));
  const [erro, setErro] = useState<ErroDeCampo | null>(null);
  const [enviando, setEnviando] = useState(false);

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

  function recusar(erroDeCampo: ErroDeCampo) {
    setErro(erroDeCampo);
    if (erroDeCampo.campo) refs[erroDeCampo.campo].current?.focus();
  }

  async function salvar(evento: React.FormEvent) {
    evento.preventDefault();
    setEnviando(true);
    setErro(null);
    // Cadastrar e editar são a mesma etiqueta inteira: `requisicaoDeSalvar`
    // escolhe método e rota.
    const { url, method } = requisicaoDeSalvar(endereco);
    try {
      const resposta = await fetch(url, {
        method,
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(valores),
      });
      // Sessão expirada leva ao Login com o caminho atual no `destino`: o
      // Comprador volta ao mesmo passo depois de entrar.
      if (resposta.status === 401) {
        router.push(paraLogin());
        return;
      }
      const corpo = await resposta.json().catch(() => null);
      if (!resposta.ok) {
        // A mensagem exibida é sempre a do envelope do servidor (AD-14): é ela
        // que fala a Voice and Tone e que nomeia o limiar, que mora na Config e
        // não no JavaScript.
        recusar(erroDeSalvar(corpo));
        return;
      }
      // Um 2xx sem `id` não é um Endereço que o checkout possa marcar: vira a
      // falha genérica no alerta, em vez de seguir com um Endereço vazio.
      if (typeof corpo?.id !== "string" || !corpo.id) {
        recusar({ campo: null, mensagem: FALHA_AO_SALVAR });
        return;
      }
      await aoSalvar(corpo as Endereco);
    } catch {
      recusar({ campo: null, mensagem: FALHA_DE_REDE });
    } finally {
      // No finally, e não no fim: o `return` do 401 pula o corpo da função, e
      // sem isto os botões ficariam desabilitados esperando uma navegação que
      // pode nem trocar a rota.
      setEnviando(false);
    }
  }

  function campoDe(campo: Campo, rotulo: string, autoComplete: string, dica?: string) {
    const comErro = erro?.campo === campo;
    const id = `${prefixo}-${campo}`;
    const idErro = `${id}-erro`;
    const idDica = `${id}-dica`;
    const descrito = [dica ? idDica : null, comErro ? idErro : null].filter(Boolean).join(" ");
    return (
      <div className="space-y-2">
        <Label htmlFor={id}>{rotulo}</Label>
        <Input
          id={id}
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
    // noValidate: a mensagem que vale é a do Go, e a bolha nativa do navegador
    // a esconderia antes de a requisição sair.
    <form className="space-y-4" onSubmit={salvar} noValidate>
      {campoDe("destinatario", "Quem recebe", "name")}
      {campoDe("cep", "CEP", "postal-code", "Oito dígitos, com ou sem hífen.")}
      {campoDe("logradouro", "Rua ou avenida", "address-line1")}
      {campoDe("numero", "Número", "address-line2")}
      {campoDe("complemento", "Complemento (opcional)", "address-line3")}
      {campoDe("bairro", "Bairro", "address-level3")}
      {campoDe("cidade", "Cidade", "address-level2")}
      {campoDe("uf", "UF", "address-level1", "Duas letras, como SP.")}

      {/* Erro sem campo — rede fora, ou o 409 do teto por Comprador, que não é
          de campo nenhum. */}
      {erro && !erro.campo && (
        <Alert variant="destructive">
          <AlertDescription>{erro.mensagem}</AlertDescription>
        </Alert>
      )}

      <div className="flex flex-wrap gap-2">
        <Button type="submit" disabled={enviando}>
          {enviando ? "Salvando…" : "Salvar Endereço"}
        </Button>
        {aoCancelar && (
          <Button type="button" variant="outline" disabled={enviando} onClick={aoCancelar}>
            Cancelar
          </Button>
        )}
      </div>
    </form>
  );
}

// O Endereço por extenso, como "Meus endereços" sempre o mostrou: o mesmo
// texto na lista, no checkout e na Revisão. As linhas são `span` em bloco, e
// não `p`, para caberem também dentro do `label` da escolha no checkout, que
// só aceita conteúdo de frase.
export function LinhasDoEndereco({ endereco }: { endereco: Endereco }) {
  return (
    <>
      <span className="block font-medium">{endereco.destinatario}</span>
      <span className="block">
        {endereco.logradouro}, {endereco.numero}
        {endereco.complemento && ` — ${endereco.complemento}`}
      </span>
      <span className="block">
        {endereco.bairro} — {endereco.cidade}/{endereco.uf}
      </span>
      <span className="text-muted-foreground block">CEP {comMascara(endereco.cep)}</span>
    </>
  );
}

export function EnderecoPorExtenso({ endereco }: { endereco: Endereco }) {
  return (
    <address className="space-y-1 not-italic">
      <LinhasDoEndereco endereco={endereco} />
    </address>
  );
}
