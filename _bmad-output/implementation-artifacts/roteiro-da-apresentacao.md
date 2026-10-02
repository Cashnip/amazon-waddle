---
title: 'Roteiro da apresentação final (15 minutos)'
created: '2026-10-01'
deck: 'https://claude.ai/artifact/CZ8eFaginyB2ca1ZbE2tp6'
fontes:
  - '../planning-artifacts/prds/prd-azamon-2026-08-14/prd.md'
  - 'ensaio-dos-roteiros.md'
  - '../planning-artifacts/architecture/architecture-azamon-2026-09-05/deck-banca.html'
---

# Roteiro da apresentação final

Quinze minutos para três eixos ao mesmo tempo: funcionalidade, arquitetura e documentação, e
apresentação ao vivo. O roteiro parte dos três roteiros de demonstração do §12 do PRD e do ensaio
da Estória 7.4 ([`ensaio-dos-roteiros.md`](ensaio-dos-roteiros.md)), mas **não os executa por
inteiro**: no ensaio, os três levaram cerca de 30 minutos. Aqui eles viram uma demonstração de 6
minutos que aproveita as esperas do próprio sistema, e o resto do tempo explica por que o que a
banca viu funciona.

O deck está no link do `deck:` acima, com a fala de cada slide nas notas do apresentador. O link
precisa de rede: **exporte o deck em PDF** antes da apresentação, porque a demonstração roda com o
Wi-Fi desligado (NFR-15).

## A estrutura

**A demonstração vem antes da arquitetura, de propósito.** A banca vê o comportamento primeiro, e
cada slide de arquitetura aponta para algo que acabou de aparecer na tela.

| Tempo | Bloco | Slides | Quem |
|---|---|---|---|
| 0:00–0:20 | Capa | `cover` | Pessoa A |
| 0:20–1:10 | A tese: maquete contra sistema, e os quatro comportamentos sem botão (FR-24, FR-34, FR-19, NFR-7) | `tese` | Pessoa A |
| 1:10–1:50 | As seis capacidades, e o que ficou de fora de propósito | `escopo` | Pessoa A |
| 1:50–3:00 | Como foi pensado e produzido: PRD → UX → arquitetura → spec → código | `processo` | Pessoa A ou B |
| 3:00–3:20 | O mapa dos cinco caminhos, e a regra dos centavos | `mapa-demo` | Pessoa B |
| 3:20–9:20 | **Demonstração ao vivo** | `demo` | Pessoa B |
| 9:20–10:05 | A máquina de estados do Pedido | `maquina` | Pessoa C |
| 10:05–11:05 | Quatro contêineres, seis módulos, a fronteira no banco | `arquitetura` | Pessoa C |
| 11:05–12:00 | A última unidade: a ordem da trava | `ultima-unidade` | Pessoa C ou D |
| 12:00–12:50 | Pagamento simulado, mas assíncrono; um relógio só | `pagamento` | Pessoa C ou D |
| 12:50–13:50 | Como sabemos que funciona | `verificacao` | Pessoa D |
| 13:50–14:35 | O que decidimos não fazer, e os limites conhecidos (**cortável**) | `limites` | Pessoa D |
| 14:35–15:00 | "O ciclo fecha." e o Pedido do caminho feliz já `ENTREGUE` | `fim` | Pessoa A |
| — | Apêndice: perguntas prováveis e quem responde | `perguntas` | todos |

As Pessoas A a D supõem um time de quatro. Com menos gente, juntem os blocos vizinhos; quem
conduz a demonstração deve ser quem mais ensaiou.

## A fala, bloco a bloco

**Tese (0:20).** "Dez telas desconectadas são uma maquete. Cinco que levam um Pedido do clique ao
`ENTREGUE` são um sistema." E o que separa as duas coisas não aparece em tela: a Reserva é
tudo-ou-nada, o pagamento expira, o Carrinho é revalidado antes do checkout, e duas pessoas não
levam a última unidade. Nenhum tem botão. Anuncie que a banca vai ver três deles.

**Escopo (1:10).** Seis capacidades, 34 requisitos funcionais e 16 não funcionais. Marketplace no
modelo de dados, sem portal do Vendedor. O que ficou de fora está registrado com o que se perdeu.
Não leia os cartões: aponte para o Pedido e diga que ele é a peça central.

**Processo (1:50).** Antes de uma linha de código, o contrato: um PRD que já trazia os três
roteiros de demonstração como requisito, a UX, a arquitetura com 20 invariantes e a spec que fatiou
tudo em 7 épicas e 52 estórias. Método BMad, com agentes de IA escrevendo, revisando e verificando;
cada épica fechou só depois do passeio de uma pessoa do time. Três regras: esqueleto vertical
primeiro, toda decisão com o porquê no memlog, e nada novo enquanto os roteiros não passassem.
**O time decide com que palavras conta o uso dos agentes** — a banca vai perguntar.

**Máquina de estados (9:20).** Histórico, cancelamento, painel e simulação de entrega são uma
máquina vista de quatro ângulos. Uma função só muda o Status, por compare-and-swap; quem chega
segundo recebe uma recusa nomeada. A Reserva acompanha a máquina e se consolida no envio, o mesmo
ponto em que o cancelamento deixa de valer (FR-28, AD-3).

**Arquitetura (10:05).** O Next.js é casca, sem regra de negócio. O Go é um monólito modular de
seis módulos, cada um com o seu schema, e chave estrangeira cruzando schema é proibida. Dois testes
vigiam a fronteira: o grafo de importação (`internal/fronteira_test.go`) e o banco. O Redis só
guarda o que expira (AD-1, AD-2, AD-10, AD-20).

**A última unidade (11:05).** O Estoque disponível é calculado: total menos Reservas ativas.
Primeiro trava as linhas de Produto `ORDER BY id FOR UPDATE`, só então soma as Reservas. Teste:
oito checkouts paralelos, um Pedido; sem o `ORDER BY`, deadlock `40P01` (NFR-7, AD-4, AD-5).

**Pagamento (12:00).** Assíncrono porque um gateway real é assíncrono. Confirmar → Provedor
Simulado → webhook → inbox → relógio. Idempotência por restrição única, só a Tentativa corrente
vale, e nenhum temporizador em memória: o tempo deriva do histórico (AD-6, AD-7, AD-8).

**Verificação (12:50).** 19 de 19 passos dos três roteiros ensaiados de clone limpo; 4 min 28 s do
`git clone` ao sistema rodando (teto: 15 min); pior p95 da busca de 8,47 ms (teto: 500 ms); 0 das
260 decisões do addendum contraditas pelo código.

**Limites (13:50).** Microsserviços são destino, não ponto de partida; Elasticsearch, Stripe e S3
ficam atrás de fronteira. Fale em voz alta de **um** limite conhecido — o cookie de Sessão
compartilhado é o mais concreto. Se o relógio já passou de 13:50, pule este slide.

**Fechamento (14:35).** "O primeiro Pedido de hoje saiu de 'aguardando pagamento' e chegou a
`ENTREGUE` sozinho enquanto a gente falava — com Estoque que baixou, um pagamento que recusou,
outro que expirou e um cancelamento que devolveu o Produto à prateleira. Esse é o ciclo fechado."

## A demonstração, minuto a minuto

Os cinco caminhos correm no mesmo banco, em sequência, sem nada reconfigurado. O desfecho do
pagamento vem dos centavos do total: `,00`–`,89` aprova, `,90`–`,94` recusa, `,95`–`,99` a
confirmação nunca chega. O Frete é inteiro (Sudeste R$ 15,00, Sul R$ 20,00), então os centavos do
total são os do Produto — **e a quantidade muda o total**: duas unidades do Andarilho recusam em
vez de expirar.

| Caminho | Produto e total | O que aparece | Requisito |
|---|---|---|---|
| 1 · Caminho feliz | Caixa de Som Portátil Maré · R$ 209,00 (Frete Sul) | `PAGO` em ~7 s sem recarregar; depois `ENTREGUE` sozinho | FR-1 a FR-33 |
| 2 · Pagamento recusado | Fone de Ouvido Bluetooth Aurora · R$ 264,90 | a recusa com o motivo; a nova Tentativa aprova | FR-27 |
| 3 · Pagamento expirado | Anotações de um Andarilho · R$ 54,95 | sai sozinho em 60 s, e o Estoque volta | FR-34 |
| 4 · Cancelamento | o Pedido da Aurora, já `PAGO` | `CANCELADO`, e a Reserva é liberada | FR-31 |
| 5 · Cancelamento recusado | o Pedido da Maré, já `ENTREGUE` | "não pode mais ser cancelado" | FR-31 |

**O truque é sobrepor as esperas.** O Pedido da Maré anda até `ENTREGUE` em ~1 min 40 s enquanto
os outros caminhos acontecem; o do Andarilho expira em 60 s enquanto a Aurora recusa, paga e é
cancelada. Todo Produto semeado tem Estoque 10, então a Reserva fica visível na Página de Produto:
"Restam 9 unidades" durante, "Em estoque" depois.

| Minuto | Perfil | O que fazer e dizer |
|---|---|---|
| 0:00 | A | Vitrine. "O Wi-Fi está desligado. Isto subiu de um `docker compose up`, com o Catálogo Semeado." |
| 0:20 | A | Cadastrar conta nova. O cadastro já abre a Sessão. |
| 0:45 | A | Buscar `mare`, sem acento: acha Maré e Marés. Faixa R$ 10–500; ordenar por menor preço. |
| 1:20 | A | Página da Caixa de Som Maré: "Vendido por Atlântico Importados" — a espinha de marketplace. |
| 1:40 | A | Adicionar 2 unidades, abrir o Carrinho, ajustar para 1: o Subtotal acompanha. |
| 2:00 | A | Checkout: Endereço de SP → Revisão com Frete Sudeste R$ 15,00. Trocar para Curitiba: Frete Sul R$ 20,00, total R$ 209,00. "Confirmar Pedido" é o único botão laranja: o passo irreversível. |
| 2:50 | A | "Aguardando pagamento" vira "Pago" em ~7 s, sem recarregar. "Vamos deixar este Pedido andar sozinho." |
| 3:00 | B | Carrinho com o Andarilho → Revisão R$ 54,95 → Confirmar. "Esta confirmação nunca vai chegar." Recarregar a aba do Andarilho: "Restam 9 unidades". |
| 3:20 | B | Adicionar a Aurora, checkout, Confirmar R$ 264,90 → "Pagamento recusado", com o motivo e as Tentativas restantes. |
| 4:00 | B | "Tentar pagar de novo" → "Pago". O histórico guarda a recusa. |
| 4:20 | B | "Cancelar Pedido" → `Dialog` → "Cancelado". A Reserva foi liberada. |
| 4:45 | B | Aba do Pedido do Andarilho: saiu sozinho, "Tempo de pagamento expirado". Recarregar a Página de Produto: "Em estoque" de novo. |
| 5:15 | C | **Opcional.** Painel do Administrador: os Pedidos por Status e as transições que são dele. |
| 5:40 | A | O Pedido da Maré em `ENTREGUE`, a linha do tempo inteira, e "não pode mais ser cancelado". |

**Se der errado.** O Pedido da Aurora já foi a `ENVIADO` antes do cancelamento? Mostre a recusa do
cancelamento — também é caminho. Atrasou? Corte nesta ordem: o painel do Administrador, a troca de
Endereço, o cadastro (use uma conta pronta). Travou? Troque para o vídeo de reserva no mesmo ponto.

A disputa pela última unidade (Roteiro C, passo 5) **não entra ao vivo** nos 15 minutos: ela fica
no slide `ultima-unidade`, com o SQL e o resultado do teste. Se sobrar tempo, ou se a banca pedir,
dá para mostrá-la com dois perfis de Comprador — só com o teste do NFR-7 verde no mesmo dia.

## Preparo

**Na véspera:**
- clone limpo, `docker compose up --build` e a demonstração inteira, cronometrada;
- **gravar um vídeo** da demonstração inteira, para servir de reserva;
- exportar o deck em PDF;
- preencher os nomes da capa e a coluna "Quem" do apêndice.

**No dia, antes de entrar na sala:**
1. Rodar o teste da NFR-7 (a invocação está em [`ensaio-dos-roteiros.md`](ensaio-dos-roteiros.md),
   seção "A condição do passo 5 do Roteiro C"). Verde no dia do ensaio não é verde no dia da
   apresentação.
2. `docker compose down -v && docker compose up --build` — nunca `up` sozinho. Confira a idade das
   imagens em `docker compose images`. O `down -v` faz a numeração recomeçar em `AZ-2026-000001`,
   que é o número do slide de encerramento.
3. Desligar o Wi-Fi da máquina.
4. Três perfis do Chrome, porque Comprador e Administrador dividem o cookie de Sessão:
   - **A** — vazio, aberto na Vitrine;
   - **B** — Comprador preparado pela tela: conta criada, Endereço de SP salvo, Andarilho no
     Carrinho; abas abertas na Página de Produto do Andarilho e da Aurora;
   - **C** (opcional) — Administrador em `/admin/entrar` → `/admin/pedidos`.
5. Zoom do navegador entre 125% e 150%, para o projetor.

Nada disso mexe no banco à mão: o perfil B é montado pela interface, como qualquer Comprador
faria.

## O que o time ainda decide

- **Nomes:** os da capa, e quem responde cada pergunta do apêndice (o §12 do PRD exige
  responsável nomeado antes da apresentação).
- **Como contar o uso de agentes de IA** no slide `processo`.
- **Se o slide `limites` fica.** Mostrar limites é coerente com a tese de honestidade do projeto,
  mas o slide é o primeiro a cortar.
- **Os 8,47 ms** do slide `verificacao` são o pior p95 medido na retrospectiva da Épica 3
  (`epic-3-retro-2026-09-19.md`); confiram as condições da medida antes de citá-la.
- **O ensaio das duas pessoas** que o §12 exige continua em aberto no Registro de ensaios de
  [`ensaio-dos-roteiros.md`](ensaio-dos-roteiros.md). Ensaiem este roteiro, não só os três do PRD:
  o tempo de 6 minutos só existe com a sobreposição das esperas.
