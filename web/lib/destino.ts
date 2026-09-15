// O `destino` é para onde a tela que levou 401 quer voltar depois do login.
// Mora aqui, e não na página, porque quem monta o link (as telas do 401) e
// quem o lê (`/entrar`) são arquivos diferentes — e é o mesmo contrato.

const LOGIN = "/entrar";

// destinoSeguro só aceita caminho relativo: o valor vem da URL, e um destino
// absoluto faria de `/entrar` um redirecionador aberto — `?destino=http://mal`
// levaria ao domínio de quem mandou o link, com a credencial acabada de
// digitar. Qualquer outra coisa vira "/".
//
// São três recusas, e as três burlam a primeira sozinhas:
//   - `//mal` é protocolo-relativo, e a barra invertida entra junto porque o
//     navegador normaliza `/\mal` para `//mal`;
//   - caractere de controle porque o navegador o descarta antes de resolver a
//     URL, e `/%09/mal` vira `//mal` depois do descarte;
//   - o próprio `/entrar` porque a página empurraria para si mesma, deixando o
//     botão preso em "Entrando…".
export function destinoSeguro(busca: string): string {
  const destino = new URLSearchParams(busca).get("destino") ?? "/";
  if (!/^\/($|[^/\\])/.test(destino)) return "/";
  if ([...destino].some((c) => c.charCodeAt(0) < 0x20 || c.charCodeAt(0) === 0x7f)) return "/";
  if (destino.split(/[?#]/)[0] === LOGIN) return "/";
  return destino;
}

// paraLogin monta o link que as telas do 401 usam. O padrão é o endereço atual
// inteiro — caminho e busca —, porque voltar a `/pedidos/x` sem a busca não é
// voltar ao ponto em que o Comprador parou. O parâmetro existe para o teste
// montar a ida e volta sem navegador.
export function paraLogin(
  caminho: string = window.location.pathname + window.location.search,
): string {
  return `${LOGIN}?destino=${encodeURIComponent(caminho)}`;
}
