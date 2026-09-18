// O fetch das telas administrativas: mesma origem, JSON no corpo quando há
// corpo, e o envelope lido sem estourar quando a resposta não é JSON. Nenhuma
// regra mora aqui (AD-10) — quem decide é o Go.
export async function pedir(url: string, method: string, corpo?: unknown) {
  const resposta = await fetch(url, {
    method,
    credentials: "same-origin",
    headers: corpo === undefined ? undefined : { "Content-Type": "application/json" },
    body: corpo === undefined ? undefined : JSON.stringify(corpo),
  });
  const json = resposta.status === 204 ? null : await resposta.json().catch(() => null);
  return { resposta, json };
}

export const FALHA_DE_REDE = "Não foi possível falar com o servidor.";
