---
id: SPEC-azamon
companions:
  - mapa-de-capacidades.md
  - ../../planning-artifacts/prds/prd-azamon-2026-08-14/prd.md
  - ../../planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md
  - ../../planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md
  - ../../planning-artifacts/architecture/architecture-azamon-2026-09-05/DIAGRAMA-MODULOS.md
  - ../../planning-artifacts/ux-designs/ux-azamon-2026-09-02/DESIGN.md
  - ../../planning-artifacts/ux-designs/ux-azamon-2026-09-02/EXPERIENCE.md
sources: []
---

> **Contrato canônico.** Esta SPEC e os arquivos em `companions:` são o contrato completo do que construir, testar e validar. Nenhum documento de entrada foi absorvido e aposentado: os seis companheiros continuam vivos e são leitura obrigatória a jusante — este kernel é o que os amarra e resolve a altitude, não um resumo que os substitui.

# Azamon — MVP

## Why

**Mandato com vitrine.** O Azamon é trabalho acadêmico de 2 a 4 pessoas em um semestre, avaliado em três eixos ao mesmo tempo: funcionalidade entregue, arquitetura e documentação, e apresentação ao vivo. O prazo e a banca são o mandato; o produto é o meio.

Isso força um reconhecimento incomum: o sistema tem **dois usuários reais e nenhum deles compra nada**. O primeiro é a Marina, compradora fictícia cuja experiência precisa ser convincente. O segundo é a banca, que julga. Servir só à Marina entrega um produto bonito que não defende suas escolhas; servir só à banca entrega um diagrama que ninguém consegue demonstrar.

A aposta que resolve o conflito: o valor de uma réplica da Amazon não está na superfície coberta, e sim no **ciclo fechado com honestidade arquitetural**. Dez telas desconectadas são uma maquete; cinco que sustentam um Pedido do clique ao `ENTREGUE` — com Estoque que baixa, pagamento que pode recusar e cancelamento que devolve o item à prateleira — são um sistema. Todo trade-off deste contrato resolve a favor do segundo.

## Capabilities

- **CAP-1 — Conta e Identidade** *(FR-1..FR-5)*
  - **intent:** Um visitante cria conta, abre e encerra Sessão, redefine senha e gerencia seus Endereços; o sistema distingue Comprador de Administrador e restringe cada recurso ao seu dono ou ao papel correto.
  - **success:** Um teste automatizado chama a API com o identificador de Pedido, de Carrinho e de Endereço de outro Comprador e recebe negação nos três (NFR-6). Credencial inválida devolve a mesma mensagem para e-mail inexistente e para senha errada.

- **CAP-2 — Catálogo e Estoque** *(FR-6..FR-11)*
  - **intent:** Qualquer visitante vê a Vitrine e a Página de Produto com o Vendedor e a disponibilidade; o Administrador gerencia Vendedores, Categorias, Produtos e Estoque; e o sistema impede a venda além do disponível.
  - **success:** Produto criado pelo Administrador aparece na Vitrine e na busca sem reiniciar a aplicação (Roteiro C, passo 1). Reduzir o Estoque total abaixo das Reservas de Estoque ativas é recusado informando quantas unidades estão comprometidas, e o disponível nunca fica negativo.

- **CAP-3 — Busca e Navegação** *(FR-12..FR-15)*
  - **intent:** Qualquer visitante encontra Produtos por termo livre sobre nome e descrição, restringe por Categoria e faixa de preço, ordena e pagina — com todo o estado da listagem na URL.
  - **success:** "cafe" encontra "café" e "50%" procura o texto literal, não devolve o catálogo inteiro. Com 5.000 Produtos semeados a busca com filtros responde em ≤ 500 ms no p95 (SM-4). Recarregar ou compartilhar a URL reproduz a mesma listagem, e a paginação nunca repete nem pula um Produto.

- **CAP-4 — Carrinho** *(FR-16..FR-19)*
  - **intent:** Um Comprador autenticado monta um Carrinho persistente entre Sessões e altera suas quantidades; o Carrinho é intenção — não congela preço nem reserva Estoque — e é revalidado contra o Catálogo antes de virar compromisso.
  - **success:** Produto que ficou indisponível desde que foi adicionado bloqueia o avanço ao checkout até ser removido ou ajustado. Produto cujo preço mudou exibe o preço novo ao lado do anterior e exige confirmação — nenhum Comprador é cobrado por um preço que não viu.

- **CAP-5 — Checkout com pagamento simulado assíncrono** *(FR-20..FR-27, FR-34)*
  - **intent:** Um Comprador escolhe o Endereço, vê o Frete e o total, confirma, e o Pedido nasce em `AGUARDANDO_PAGAMENTO` com Reserva de Estoque atômica sobre todos os itens; o resultado do pagamento chega depois, na tela do Pedido em processamento, e a recusa, a nova tentativa e a expiração são caminhos de primeira classe, não erros.
  - **success:** O Roteiro B inteiro passa em ambiente limpo — recusa com o Estoque voltando na Vitrine, nova tentativa aprovada, e um Pedido saindo sozinho de `AGUARDANDO_PAGAMENTO` por tempo esgotado. Duas criações simultâneas para a última unidade produzem exatamente um Pedido (NFR-7). A mesma confirmação recebida duas vezes avança o Pedido uma vez só (NFR-12). Trocar o Provedor Simulado por outra implementação não toca checkout, Pedido nem máquina de estados (SM-5).

- **CAP-6 — Pedido: máquina de estados, pós-venda e operação** *(FR-28..FR-33)*
  - **intent:** Todo Pedido tem exatamente um Status e só transita pelas transições declaradas, registrando cada uma; o Comprador consulta histórico e detalhe e cancela antes do envio, o Administrador opera a lista, e a simulação de entrega avança sozinha — quatro ângulos da mesma máquina.
  - **success:** O Roteiro C passa: cancelar em `SEPARANDO` devolve as unidades à Vitrine no mesmo instante, cancelar em `ENVIADO` é recusado inclusive por chamada direta à API, e mover um `CANCELADO` para `SEPARANDO` é recusado informando quais transições são permitidas. Reiniciar o sistema não congela nenhum Pedido em andamento. Dado um número de Pedido, a sequência de eventos que levou ao estado atual é reconstruível (NFR-9).

- **CAP-7 — Ambiente em um comando** *(NFR-1, NFR-15)*
  - **intent:** O sistema completo — aplicação, banco, cache e Catálogo Semeado — sobe com um único comando em contêineres a partir do repositório limpo, e funciona sem acesso à internet.
  - **success:** SM-3 — um integrante que nunca rodou o projeto chega ao sistema funcionando em ≤ 15 minutos seguindo apenas o README. Com a rede desconectada, o Roteiro A percorre inteiro, imagens e fonte incluídas. Subir duas vezes produz exatamente os mesmos Vendedores, Categorias e Produtos.

- **CAP-8 — Entregáveis de documentação** *(§6.1 do PRD, SM-7)*
  - **intent:** A banca encontra, no dia da entrega, um README que leva do clone ao sistema rodando, o diagrama dos seis módulos com o que atravessa cada fronteira, e o addendum mantido vivo com toda decisão de arquitetura tomada durante a construção.
  - **success:** SM-7 — os três existem e **não mentem**: o README passa a mesma verificação da SM-3, o diagrama corresponde à decomposição real do código, e percorrer as decisões do addendum contra o código não encontra nenhuma contradita.

## Constraints

- **Precedência em conflito.** O PRD decide **requisito**; a espinha de arquitetura decide **mecanismo**; `DESIGN.md` e `EXPERIENCE.md` decidem **design e comportamento**; o addendum guarda o **porquê**. Mock, wireframe e material importado perdem para as espinhas.
- **Stack imposta.** Go, PostgreSQL, Redis e Docker obrigatório vieram do usuário; Next.js 16 sobre React 19 foi decidido em 2026-09-06. Versões exatas na tabela Stack da espinha, verificadas na web — a linha 16.3.x do Next.js é o patch das duas RCE críticas de 25/08/2026, não preferência.
- **Monólito modular com seis módulos canônicos** — identidade, catálogo, busca, carrinho, pedido, pagamento — e não se negocia um sétimo. Microsserviços é destino declarado, não ponto de partida.
- **Nenhum dado de cartão entra no sistema, nem no Provedor Simulado.** Não pede número, não gera fictício, não guarda nada parecido com um.
- **O pagamento confirma fora da requisição do checkout**, por webhook, com idempotência por chave e prazo de expiração — porque é assim que um gateway real se comporta, e é o que faz a virada ser troca de implementação em vez de reescrita.
- **Nenhum estado é alcançado por temporizador em memória.** Expiração da Tentativa de Pagamento e simulação de entrega derivam do histórico de transições; reiniciar o contêiner não congela Pedido nenhum.
- **Dinheiro é inteiro de centavos ponta a ponta.** `float` e `numeric` em valor monetário são proibidos em qualquer camada, e o total exibido é sempre a soma exata das parcelas exibidas.
- **Nada de rede externa em tempo de execução.** Sem Google Fonts, sem `@import` remoto, sem CDN: a demonstração roda com a rede desconectada, e esse é o erro mais fácil de cometer.
- **Autorização no servidor, por dono do recurso**, verificada na mesma consulta que carrega o dado. Esconder o link na interface não é a proteção.
- **Disciplina de glossário.** Os 20 termos do §3 do PRD são usados literalmente em código e em tela — nunca "compra" no lugar de **Pedido**, nunca "item" no lugar de **Produto**. Sinônimo é defeito, não estilo.
- **Esqueleto vertical antes de alargar qualquer camada.** Um Produto, um Comprador, um Pedido que nasce, é pago e chega a `ENTREGUE`, com telas feias, antes de qualquer ordem por dependência. A ordem por dependência é correta e perigosa se seguida sozinha.
- **Custo zero de infraestrutura.** Tudo local, em contêineres, sem serviço pago e sem conta em nuvem. Nada é publicado, e não existe ambiente de produção — um terceiro ambiente seria arquitetura de mentira.
- **Português do Brasil e BRL** na interface e nos documentos; commits em inglês. Sem multi-idioma, sem multi-moeda, sem cupom e sem imposto destacado.
- **Piso WCAG 2.2 AA completo**, e de 360 px a 1440 px sem rolagem horizontal nem elemento inacessível. A verificação mínima é percorrer a UJ-1 inteira sem tocar no mouse.
- **Capacidade fixa: 2 a 4 pessoas, um semestre.** Quando o prazo aperta, corta-se na ordem declarada do §6.3 do PRD, não pelo que estiver mais difícil na semana — e FR-19, FR-24, FR-34 e o teste de concorrência do NFR-7 nunca são cortados, por mais invisíveis que sejam.
- **Verde e laranja têm sentido fechado.** Verde é Estoque disponível e `ENTREGUE`; laranja aparece **uma vez por fluxo**, no passo irreversível. Uso decorativo apaga o único mecanismo semântico de cor do sistema.

## Non-goals

- **Não é uma loja real.** Nenhum pagamento real, nenhuma entrega real, nenhum dado pessoal real.
- **Não vira portal de vendedor.** O Vendedor existe no modelo de dados e é dono do Produto, mas não acessa o sistema. Sem auto-cadastro, moderação de anúncio, painel de vendas ou repasse financeiro.
- **Não é um motor de busca.** Sem relevância por pontuação, correção ortográfica, facetas com contagem ou sugestão enquanto digita.
- **Não tem Carrinho anônimo nem fusão no login.** Corte consciente: elimina uma classe inteira de casos de borda num sistema sem visitantes reais.
- **Não tem conteúdo social.** Sem avaliações, notas, perguntas e respostas, comentários ou lista de desejos.
- **Não tem motor de promoções.** Sem cupom, desconto progressivo, frete promocional negociado ou fidelidade.
- **Não tem pós-venda além do cancelamento.** Sem devolução, reembolso, troca ou abertura de chamado.
- **Não divide o Pedido por Vendedor.** Um Pedido é único mesmo com Produtos de Vendedores diferentes, e o Status é do Pedido inteiro.
- **Não notifica por e-mail.** Não existe serviço de e-mail: a linha do tempo do Pedido cobre a necessidade, e o token da FR-3 sai no log estruturado.
- **Não tem upload de imagem de Produto.** A imagem é URL relativa para arquivo estático servido pelo Go; upload entra junto com o armazenamento de objetos.
- **Não é aplicativo móvel.** Web responsivo apenas, sem nativo e sem PWA instalável.
- **Não tem modo escuro.** A referência não tem, e manter dois temas dobra a superfície de verificação de contraste sem ganhar nada na banca.
- **Não é publicado.** Roda local na demonstração; sem requisito de disponibilidade, domínio ou certificado.

## Success signal

**SM-1** — os três roteiros de demonstração do §12 do PRD executam de ponta a ponta em ambiente limpo, sem intervenção manual no banco e sem erro visível. Alvo: 3 de 3. O Roteiro B é o que separa maquete de sistema: pagamento recusado com o Estoque voltando na Vitrine em outra aba, nova tentativa aprovada, e um Pedido que sai sozinho de `AGUARDANDO_PAGAMENTO` por tempo esgotado.

**SM-6** — marco intermediário: o Roteiro A executa de ponta a ponta **até a metade do semestre**, ainda que com telas cruas e catálogo mínimo. É a métrica que detecta cedo o único modo de falha capaz de anular o trabalho inteiro — chegar em dezembro com o ciclo aberto. Se não for atingida no prazo, a resposta é cortar escopo, não acelerar.

**Contra-métrica declarada:** nenhuma funcionalidade nova entra enquanto os três roteiros não passarem. Mais telas com o ciclo quebrado vale menos que o ciclo fechado.

## Assumptions

- As 13 suposições do produto ficam onde estão, no §16 do PRD, **ordenadas por risco e nunca renumeradas** — referências a "suposição N" já quebraram uma vez por causa de uma reordenação.
- Os intervalos de consulta em intervalo — 3 s na tela do Pedido em processamento, 10 s no Detalhe do Pedido e na Tabela de Pedidos do Administrador — não vêm de requisito nenhum. Foram fixados pela `EXPERIENCE.md` e endossados pelo `AD-18`.
- A família tipográfica concreta ainda não foi escolhida: qualquer sans neutra auto-hospedada serve, e o apelido `Azamon Sans` vale de qualquer jeito.

## Open Questions

- 🔴 **Existe rubrica escrita ou enunciado formal do professor?** Dono: Sung. Trava quatro suposições do §16 do PRD — Portal do Vendedor fora do MVP (n.º 1), Pedido não dividido por Vendedor (n.º 2), ausência de verificação de e-mail (n.º 7) e ausência de devolução e reembolso (n.º 8). Não são quatro riscos independentes: são um só com quatro sintomas, e o enunciado resolve os quatro de uma vez. **Não impede começar a construir** — nenhuma das quatro toca o esqueleto vertical. Ao obter a resposta: `bmad-prd` em modo atualização, depois a espinha, depois esta SPEC.
- 🟡 **O Portal do Vendedor é mesmo fase 2, ou foi omissão?** Adiada, condicionada à pergunta acima. Reversão barata: o Item de Pedido já congela o Vendedor e o `AD-11` já obriga verificação por dono do recurso, que é o predicado que falta.

*Quatro verificações de dez minutos — checagens, não decisões — pertencem ao passo 0 e vivem nas Questões em Aberto da espinha: o `Set-Cookie` do Go atravessando o `rewrites()` do Next, o sqlc analisando `DEFAULT uuidv7()`, o shadcn/ui instalando limpo em Next 16 + React 19, e o p95 do NFR-4 com 5.000 Produtos. Nenhuma bloqueia começar; a primeira é a que mais custa se for descoberta tarde.*
