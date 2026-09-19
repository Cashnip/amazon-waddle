// O Endereço do lado do navegador (FR-5), extraído de "Meus endereços" na 5.2
// para o checkout usar o mesmo formulário. Nenhuma regra mora aqui (AD-10):
// formato do CEP, siglas da UF, obrigatórios e tetos são do Go. O que há é a
// forma da resposta e a máscara de exibição.

export const CAMPOS = [
  "destinatario",
  "cep",
  "logradouro",
  "numero",
  "complemento",
  "bairro",
  "cidade",
  "uf",
] as const;
export type Campo = (typeof CAMPOS)[number];

export type Endereco = { id: string } & Record<Campo, string>;

export const VAZIO: Record<Campo, string> = {
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
export function comMascara(cep: string): string {
  return cep.length === 8 ? `${cep.slice(0, 5)}-${cep.slice(5)}` : cep;
}

// Cadastrar e editar são a mesma etiqueta inteira: muda o método e o endereço
// da rota, e mais nada. `endereco` nulo é cadastro.
export function requisicaoDeSalvar(endereco: Endereco | null): { url: string; method: "POST" | "PUT" } {
  return endereco
    ? { url: `/api/v1/enderecos/${encodeURIComponent(endereco.id)}`, method: "PUT" }
    : { url: "/api/v1/enderecos", method: "POST" };
}

export const FALHA_AO_SALVAR = "Não foi possível salvar o Endereço.";

// O erro de um salvar recusado, lido do envelope (AD-14): a mensagem é sempre a
// do servidor, e o campo é filtrado contra CAMPOS — um nome que a tela não
// conhece vira `null` e cai no alerta, em vez de sumir da tela inteira.
export function erroDeSalvar(corpo: unknown): { campo: Campo | null; mensagem: string } {
  const erro = (corpo as { erro?: { mensagem?: unknown; dados?: { campo?: unknown } } } | null)?.erro;
  const mensagem = typeof erro?.mensagem === "string" && erro.mensagem ? erro.mensagem : FALHA_AO_SALVAR;
  return { campo: CAMPOS.find((c) => c === erro?.dados?.campo) ?? null, mensagem };
}
