# Epic 4 Context: Carrinho

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

Um Comprador autenticado monta um Carrinho que sobrevive à Sessão, ajusta as quantidades e fica sabendo **antes** de qualquer surpresa. O Carrinho é intenção, não compromisso: não congela preço e não reserva Estoque. O compromisso só acontece na criação do Pedido (Épica 5). É essa distinção que impede um Carrinho abandonado de segurar o Estoque da loja. A FR-19 (revalidação) está na lista do que nunca se corta.

## Stories

- Story 4.1: Carrinho exclusivo de Comprador autenticado
- Story 4.2: Adicionar Produto ao Carrinho
- Story 4.3: Alterar quantidade, remover Item e esvaziar o Carrinho
- Story 4.4: Revalidação do Carrinho na abertura 🔒
- Story 4.5: A tela do Carrinho não promete o que o sistema não faz

## Requirements & Constraints

- **FR-16:** só Comprador autenticado tem Carrinho, e cada Comprador tem exatamente um, persistente entre Sessões e dispositivos. Não há Carrinho anônimo nem fusão no login (decisão do §6.2 do PRD e do addendum §7). Quando o visitante tenta adicionar, vai ao Login e volta à Página de Produto de origem com o Item de Carrinho já criado.
- **FR-17:** adicionar um Produto que já está no Carrinho soma à quantidade do Item existente; nunca aparecem duas linhas do mesmo Produto. A quantidade é inteira, 1 ≤ q ≤ teto por Item (padrão 10, vindo da configuração), e esse teto é verificado independentemente do Estoque. Quantidade acima do Estoque disponível é recusada informando o disponível. Produto indisponível ou invisível não entra.
- **FR-18:** quantidade zero remove o Item de Carrinho. O total é recalculado a cada alteração pelos preços atuais do Catálogo. Esvaziar é uma ação só, com confirmação.
- **FR-19 🔒:** ao abrir o Carrinho, o sistema revalida cada Item contra o Catálogo. Mudança de preço aparece e pede confirmação. Produto indisponível, desativado ou de Vendedor desativado recebe o mesmo sinal e bloqueia o avanço. A metade que acontece na entrada do checkout, junto com o recálculo do Frete, fica para a Épica 5.
- **NFR-6:** o teste automatizado de acesso a recurso de outro Comprador ganha o Carrinho como recurso coberto.
- **NFR-8:** cada FR desta épica precisa de ao menos um teste que falhe se a consequência declarada quebrar.
- **NFR-13 / NFR-14 / NFR-16:** valores em centavos inteiros, e o total exibido é a soma exata das parcelas exibidas. A quantidade tem teto verificado no servidor, e o teto vem da configuração `AZAMON_`.

## Technical Decisions

- **Posse e dados (AD-2):** o módulo `internal/carrinho` é dono do schema `carrinho` (`carrinho`, `item_carrinho`). O Carrinho vive no PostgreSQL, não no Redis, então sobrevive a reinício e não expira sozinho. Nenhuma FK cruza schema: `item_carrinho.produto_id` e `carrinho.comprador_id` são `uuid` sem FK. O Item de Carrinho congela só `preco_visto_centavos`, que é o preço na última alteração e permite dizer "de X para Y".
- **`carrinho` não conhece `identidade` (AD-11):** `api/` resolve a Sessão e injeta o `comprador_id`. A posse é verificada dentro de `carrinho`, na mesma consulta que carrega o dado. Carrinho de outro Comprador devolve o mesmo erro de inexistente.
- **Leitura do Catálogo (AD-5, AD-19):** preço e visibilidade vêm só pela interface pública de `catalogo` (`Disponivel` em lote e o predicado de "Produto visível"), nunca por consulta ao schema alheio. `Disponivel` só orienta e devolve `0` para Produto invisível ou inexistente. Quem decide é `Reservar`, sob bloqueio, na criação do Pedido. Divergência entre o número exibido e o aceito é comportamento projetado, não defeito. O Carrinho não chama `Reservar`.
- **Revalidação sem escrita (AD-17):** `carrinho.Itens` é leitura pura e nunca grava `preco_visto_centavos`. Quem registra que o Comprador viu a mudança é o checkout, por `carrinho.ConfirmarPrecoVisto` (estória 5.5). `carrinho` não calcula Frete; a distância até a isenção usa só o limiar da configuração.
- **Esvaziamento pelo Pedido:** `pedido.Criar` chama `carrinho.Esvaziar(ctx, tx, compradorID, itemIDs)` na mesma transação, depois que `Reservar` dá certo. É o único ponto em que outro módulo escreve no Carrinho. Cancelar um Pedido não devolve nada ao Carrinho.
- **Dinheiro (AD-9):** `int64` com sufixo `_centavos`, sem `float` nem `numeric`. A formatação em `R$` acontece só no navegador, com `Intl.NumberFormat('pt-BR')`.
- **Erros (AD-14):** o módulo exporta erros sentinela e só `internal/plataforma/erro` traduz para HTTP. O envelope `{"erro":{"codigo","mensagem","dados","correlacao"}}` leva dado junto: a recusa por Estoque carrega `disponivel` e `solicitado`, que a tela usa na mensagem.

## UX & Interaction Patterns

- **Contador na barra superior:** soma unidades, não linhas, e atualiza sem recarregar. Sem Sessão, o ícone aparece sem contador e leva ao Login. Não existe na área administrativa.
- **Botão Adicionar ao Carrinho:** espera o servidor, sem atualização otimista. Quando dá certo, atualiza o contador. A recusa por Estoque mostra o número disponível.
- **Linha de Item de Carrinho (UX-DR8):** feita com `Input` e `Button`, separada por `Separator`, nunca por `Card` aninhado. A quantidade é editada na própria linha com atualização otimista, a **única** permitida no sistema (UX-DR18); se falhar, reverte e explica. Remover não pede confirmação. Esvaziar pede, num `Dialog` de um nível só, e `Esc` fecha.
- **Estados (UX-DR12):** Carrinho vazio mostra "Seu Carrinho está vazio." com uma ação só, voltar à Vitrine. Preço mudado abre um `Alert` com o preço anterior e o atual de cada Item e exige confirmação. Unidade indisponível ou Vendedor desativado abre um `Alert` que **bloqueia** o avanço e oferece remover ou ajustar ali mesmo. Confirmar não substitui o bloqueio.
- **Promessas da tela (UX-DR17):** a composição monetária é só o subtotal, sem Frete e sem total estimado. A distância até o Frete grátis aparece em valor absoluto ("Faltam R$ 42,00 para o Frete grátis."), sem barra de progresso. O número exibido é sempre o Estoque disponível. O Comprador nunca vê o termo Reserva de Estoque, nem textos como "preço garantido" ou "reservado para você", e não há urgência fabricada. Valores usam `tabular-nums`.
- **Glossário (UX-DR16):** os termos aparecem literalmente na tela, "Produto" e nunca "item", "Pedido" e nunca "compra". O Carrinho atualiza só por ação do usuário, sem consulta em intervalo.

## Cross-Story Dependencies

- **Das épicas anteriores:** a Sessão, o Login com retorno ao ponto de partida e o teste do NFR-6 sobre o Endereço vêm da Épica 2. `Disponivel`, o predicado de visibilidade e o botão Adicionar ao Carrinho na Página de Produto e no Cartão de Produto vêm da Épica 3. O caminho visitante → Login → volta com o Item criado, deixado aberto na 3.6, se fecha na 4.1.
- **Dentro da épica:** a 4.1 cria a migração e a posse, e as outras dependem dela. A 4.2 e a 4.3 escrevem `preco_visto_centavos`, que a 4.4 lê. A 4.5 é a camada de texto e composição sobre a tela da 4.3 e da 4.4.
- **Para a Épica 5:** a 5.5 completa a FR-19 na entrada do checkout com `ConfirmarPrecoVisto` e o recálculo do Frete. `pedido.Criar` consome `carrinho.Esvaziar`. Essas assinaturas precisam existir e ficar estáveis ao fim desta épica.
