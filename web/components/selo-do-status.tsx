import { Badge } from "@/components/ui/badge";
import { aparenciaDoSelo, rotuloDoStatus } from "@/lib/pedido";

// A forma única dos sete Status (6.7, UX-DR10): o mesmo selo em Meus pedidos,
// no Detalhe do Pedido e na Tabela do Administrador. O texto vem de
// `rotuloDoStatus` e a aparência — variante e classes de cada tom — de
// `aparenciaDoSelo`, em `lib/pedido.ts`, onde o `node --test` a prende. Aqui
// só se desenha: o `Badge` funde o `className` sobre a base pelo `cn`. Sem
// ícone: o texto já diz o Status.
export function SeloDoStatus({ status }: { status: string }) {
  const { variant, className } = aparenciaDoSelo(status);
  return (
    <Badge variant={variant} className={className}>
      {rotuloDoStatus(status)}
    </Badge>
  );
}
