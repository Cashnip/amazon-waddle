import { cn } from "cn";

// O valor viaja em centavos inteiros de ponta a ponta (AD-3); o `R$` é escrito
// aqui, no front, e em nenhum outro lugar. Os dois papéis tipográficos
// monetários trazem o tabular-nums junto.
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
