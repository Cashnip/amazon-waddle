// O indicador dos dois passos do checkout, Endereço → Revisão. Frete e
// pagamento são blocos da Revisão, e não passos. Cada passo é uma rota
// própria, então "um passo por tela" de 360 a 639 px vale por construção
// (UX-DR15). O passo atual é marcado por `aria-current`, e não por cor: nem
// verde nem laranja aqui.
const PASSOS = ["Endereço", "Revisão"] as const;

export function EtapasDoCheckout({ atual }: { atual: (typeof PASSOS)[number] }) {
  return (
    <nav aria-label="Passos do checkout">
      <ol className="text-muted-foreground flex flex-wrap items-center gap-2 text-sm">
        {PASSOS.map((passo, i) => (
          <li key={passo} className="flex items-center gap-2">
            {i > 0 && <span aria-hidden="true">→</span>}
            <span
              aria-current={passo === atual ? "step" : undefined}
              className={passo === atual ? "text-foreground font-medium underline underline-offset-4" : undefined}
            >
              {i + 1}. {passo}
            </span>
          </li>
        ))}
      </ol>
    </nav>
  );
}
