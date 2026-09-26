import { cn } from "cn";

// O valor viaja em centavos inteiros de ponta a ponta (AD-9); o `R$` é escrito
// no front, aqui e em `formatarPreco` (web/lib/preco.ts), que as telas de
// texto corrido usam. Os dois papéis tipográficos monetários trazem o
// tabular-nums junto.
export function Preco({ centavos, className }: { centavos: number; className?: string }) {
  const reais = Math.floor(centavos / 100);
  const resto = String(centavos % 100).padStart(2, "0");
  return (
    <p className={cn(className)}>
      <span className="preco">R$ {reais.toLocaleString("pt-BR")}</span>
      <span className="preco-centavos align-super">{resto}</span>
    </p>
  );
}
