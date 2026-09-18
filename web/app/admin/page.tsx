import { redirect } from "next/navigation";

// `/admin` não tem tela própria: o primeiro destino da Navegação
// administrativa é Vendedores, e a casca de lá cuida de quem não tem Sessão.
export default function Admin() {
  redirect("/admin/vendedores");
}
