// sair() é a única saída de Sessão do lado do navegador — usada pelo Menu da
// conta (dropdown) e pelo botão próprio do Perfil (2.6). Fica num módulo à
// parte, e não dentro de menu-da-conta.tsx: duas cópias do DELETE e do reload
// divergiriam na primeira mudança, e menu-da-conta.tsx é um componente de UI
// de dropdown, não o lugar natural para quem só quer chamar sair().
//
// O DELETE é idempotente e responde 204 mesmo sem Sessão — não há erro a
// exibir —, e recarregar é o que faz a casca inteira, as oito telas, refletir
// a saída.
export async function sair() {
  await fetch("/api/v1/sessao", { method: "DELETE", credentials: "same-origin" }).catch(() => {});
  window.location.reload();
}
