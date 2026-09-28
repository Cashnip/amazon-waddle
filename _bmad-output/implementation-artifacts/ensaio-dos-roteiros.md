---
title: 'Ensaio dos três roteiros de demonstração (Estória 7.4)'
created: '2026-09-28'
baseline_commit: 'ddcc36139e47b63539454dd8a4d216e73b74a8aa'
spec: 'spec-7-4-os-tres-roteiros-ensaiados-a-partir-de-clone-limpo.md'
---

# Ensaio dos três roteiros

A prova da SM-1. Os três roteiros do §12 do PRD, executados de ponta a ponta **no navegador**,
a partir de **clone limpo** de `main`, sem `psql` e sem nada reconfigurado entre roteiros. Um
passo por passo do §12, com o FR que o sustenta, o dado de demonstração que provoca o desfecho,
o que tem de aparecer na tela e a evidência do que **esta execução** viu — com o status e o
horário que a própria tela mostrou. Passo que não foi executado aparece como **não ensaiado**,
nunca como aprovado.

O README continua sendo o caminho do clone ao sistema rodando; este arquivo é o documento de
apresentação, ao lado do [`percurso-do-addendum.md`](percurso-do-addendum.md), porque é prova de
métrica. Defeito achado aqui **não foi corrigido**: a Épica 7 verifica, não constrói, e corrigir
dentro do ensaio destruiria a evidência. Cada um virou entrada em
[`deferred-work.md`](deferred-work.md) com o roteiro e o passo.

## Contagem

| Veredito | Quantos | Quais |
|---|---|---|
| **aprovado** | **19** | Roteiro A 1–9, Roteiro B 1–5, Roteiro C 1–5 |
| **reprovado** | **0** | — |
| **não ensaiado** | **0** | — |
| **Total dos três roteiros** | **19** | |
| *(dos 19 acima)* aprovado **só na segunda tentativa** | 1 | **Roteiro C passo 3** — a primeira tentativa perdeu a janela de ~30 s e o Pedido **AZ-2026-000004** chegou a `ENTREGUE` antes do cancelamento (`PAGO` 11:35:36 → `ENTREGUE` 11:36:52), então foi descartado e o passo foi refeito com **AZ-2026-000005** |
| *(dos 19 acima)* aprovado **com contorno** | 4 | **Roteiro C passos 1 a 4** — dois perfis de navegador separados, um por papel: o contorno está no defeito 1 e não está no §12 |

Os dois últimos números não somam nem subtraem nada dos 19: são ressalvas sobre **como** quatro
desses passos passaram. A SM-1 é sobre conduzir os roteiros ao vivo, e de uma vez: um passo que
só fecha na segunda tentativa passa, mas avisa o apresentador de onde o roteiro é apertado, e um
passo que só fecha com contorno não é um passo que funciona como está escrito.

Nenhum erro visível em passo nenhum: nenhuma notificação com identificador de correlação
apareceu em nenhuma das três execuções. As recusas que apareceram são de regra de domínio
(pagamento recusado, expiração, Estoque insuficiente, cancelamento fora da janela), e cada uma
era o desfecho esperado do seu passo.

Fora dos 19 passos, este ensaio também viu: **Meus pedidos vazio** ("Você ainda não fez nenhum
Pedido.", conta nova em 11:43), o **bloco neutro com o nome do Produto** no lugar da imagem
ausente (o Produto criado no Roteiro C passo 1 nasceu sem imagem), e a **recusa do cancelamento
pelo servidor na corrida** (Roteiro C passo 4, variante *b*). O que continua **sem ver na tela**
está listado em [O que este ensaio não cobriu](#o-que-este-ensaio-não-cobriu).

## O ambiente deste ensaio

Clone limpo em `C:\dev\ensaio74`, fora do repositório de trabalho, pelo comando da §1 do README:

```bash
git clone -c core.longpaths=true https://github.com/Cashnip/amazon-waddle ensaio74   # 4 s
cd ensaio74 && docker compose up --build -d                                          # 44 s
```

`HEAD` do clone: `ddcc361`, o mesmo `baseline_commit` desta estória. `docker compose up --build`,
nunca `up` sozinho — e conferido em `docker compose images`: as imagens `ensaio74-azamon` e
`ensaio74-web` tinham **33 s e 27 s** quando o ensaio começou, não horas. Os quatro serviços no
ar — `azamon`, `postgres` e `redis` `healthy`, e o quarto é o `web`, que **não tem healthcheck**
(por isso se confere pela porta 3000, como a §1 do README avisa); `GET /api/v1/saude` devolveu
`{"status":"ok"}` e a porta 3000 respondeu 200. Vitrine com o Catálogo Semeado: **1–20 de 50 resultados**, 5 Categorias.

44 s porque as imagens base e o cache de construção já existiam nesta máquina — não é medida do
NFR-1, que é o **Registro de subidas** do README (§1) e continua sendo lá que se mede.

**Nenhum reinício entre roteiros.** Não houve `down -v` nenhum entre A, B e C: os três correram
sobre o mesmo banco, na sequência, com os Pedidos de um visíveis para o outro. Nada foi
reconfigurado: o `.env` versionado subiu como está, e todo desfecho de pagamento veio da escolha
de Produto e quantidade.

Navegador: Chrome 153, três perfis separados — um para o Comprador, um para o Administrador e um
para o segundo Comprador do passo 5 do Roteiro C. Os perfis separados **não são conforto**: são a
armadilha 2 mais abaixo.

**Desmontagem.** Feita ao fim do ensaio, no clone: `docker compose down -v` (contêineres, rede
`ensaio74_default` e o volume `ensaio74_azamon-postgres` removidos), `docker rmi` das duas imagens
do ensaio (`ensaio74-azamon:latest` e `ensaio74-web:latest`), o diretório `C:\dev\ensaio74`
apagado e os três Chrome do ensaio encerrados. Não sobrou contêiner, imagem, volume nem perfil de
navegador do ensaio; `postgres:18.6` e `redis:8.10.1`, que são imagens públicas e não do ensaio,
ficaram como estavam.

---

## Roteiro A — o caminho feliz (UJ-1, UJ-2)

Dado de demonstração: **Caixa de Som Portátil Maré**, R$ 189,00, **1 unidade**, CEP `01310-100`
(SP) → Frete R$ 15,00 → **total R$ 204,00**, centavos `,00`: o Provedor Simulado **aprova**.

| # | FR | Dado da demonstração | O que tem de aparecer | Evidência desta execução | Veredito |
|---|---|---|---|---|---|
| 1 | NFR-1 | `docker compose up --build`, sem argumento a mais | os quatro serviços no ar e a Vitrine com o Catálogo Semeado | 11:04:40 os quatro no ar, três `healthy`; `saude` `{"status":"ok"}`; Vitrine "1–20 de 50 resultados", 5 Categorias na barra | **aprovado** |
| 2 | FR-1, FR-2 | conta nova `ensaio74@azamon.test` | cadastro aceito e Sessão de Comprador | 11:14:28 `/cadastrar` aceitou e levou à Vitrine — **o cadastro já abre a Sessão**: `POST /api/v1/compradores` chama `abrirSessao` (`api/comprador.go:58`), e é isso que `web/app/cadastrar/page.tsx:76` documenta. Esta execução ainda passou por `/entrar` às 11:14:32 porque o ensaio queria ver a porta da FR-2 **também**, e não porque faltasse Sessão: a barra voltou com "Olá, Ensaio 7.4" e o contador do Carrinho em 0, e 11:14:34 `/perfil` abriu com `ensaio74@azamon.test`. Quem apresentar pode ir do cadastro direto à compra | **aprovado** |
| 3 | FR-12, FR-13, FR-14 | termo `mare`, sem acento; faixa R$ 10 a R$ 500; ordenação por preço | os Produtos com "Maré"/"Marés" apesar do acento errado, a faixa aplicada e a ordem trocando | 11:15:48 `?termo=mare` → "1–2 de 2 resultados para 'mare'": Caixa de Som Portátil **Maré** R$ 189,00 e O Silêncio das **Marés** R$ 49,90. 11:15:50 faixa → fichas "A partir de R$ 10,00" e "Até R$ 500,00", 2 resultados. 11:15:53 "Preço: menor primeiro" → O Silêncio das Marés **antes** da Caixa de Som | **aprovado** |
| 4 | FR-7 | a Página de Produto da Maré | o Vendedor na página — é aqui que se explica a espinha de marketplace | 11:16:08 `/produtos/a0ae8ff1-…`: breadcrumb "Vitrine › Eletrônicos", **"Vendido por Atlântico Importados"**, R$ 189,00, "Em estoque" e a Caixa de compra com Quantidade e "Adicionar ao Carrinho" | **aprovado** |
| 5 | FR-17, FR-18 | adicionar **2 unidades**, ajustar para **1** no Carrinho | o Carrinho somando, e o Subtotal seguindo o ajuste | 11:18:12 com 2 un.: contador da barra **2**, Carrinho "R$ 189,00 cada", linha R$ 378,00, "Subtotal: R$ 378,00". 11:19:05 ajustado a 1: contador **1**, "Subtotal: R$ 189,00" e "Faltam R$ 110,00 para o Frete grátis." | **aprovado** |
| 6 | FR-20, FR-21, FR-22 | Endereço SP `01310-100`; depois Curitiba `80010-000` (Sul); depois SP de volta | Endereço → Revisão, e o **Frete recalculado** ao trocar de região | 11:19:33 Endereço SP salvo no passo "1. Endereço". 11:19:44 Revisão: Subtotal R$ 189,00 · **Frete (Sudeste) R$ 15,00** · Total **R$ 204,00**. 11:20:51 "Trocar o Endereço" → Endereço de Curitiba/PR cadastrado e escolhido; 11:21:04 Revisão: **Frete (Sul) R$ 20,00** · Total **R$ 209,00**. 11:21:20 de volta ao de SP: **R$ 15,00** · **R$ 204,00** | **aprovado** |
| 7 | FR-23, FR-25, FR-26 | "Confirmar Pedido" com o total em R$ 204,00 | o Pedido em processamento virar `PAGO` **sem recarregar** | 11:21:34 Confirmar → `/pedidos/01a0e864-…`, **AZ-2026-000001**, "Status: Aguardando pagamento", "Tempo restante para o pagamento: 0:59", contador do Carrinho zerado. Sem recarregar, a mesma tela passou a "Status: **Pago**" às 11:21:42 — o histórico diz `Aguardando pagamento` 11:21:32 → `Pago` **11:21:39**, **7 s** | **aprovado** |
| 8 | FR-29, FR-30 | "Meus pedidos" e o Detalhe | o **preço praticado** e a **linha do tempo** | 11:21:59 `/pedidos`: "AZ-2026-000001 · 28/09/2026 · Pago · R$ 204,00 · Página 1 de 1 · 1 Pedido". 11:22:02 Detalhe: Itens do Pedido "Caixa de Som Portátil Maré — **1 × R$ 189,00**" e Histórico com cada transição e seu "antes:" | **aprovado** |
| 9 | FR-33 | nada — a varredura é que move | `ENTREGUE` sozinho, durante o resto | a mesma tela, sem recarregar: `Separando` **11:22:10**, `Enviado` **11:22:41**, `Entregue` **11:23:12** — 30 s por etapa, **1 min 40 s** do Confirmar ao `ENTREGUE`. Ao fim: "Este Pedido já foi entregue e não pode mais ser cancelado." | **aprovado** |

Os tempos batem com os que o README §2 já registrava pela API (`PAGO` em 7 s, `ENTREGUE` em
1 min 39 s). Este ensaio os viu **na tela**, que é o que a SM-1 pede.

---

## Roteiro B — o caminho triste

Dado de demonstração dos passos 1–4: **Fone de Ouvido Bluetooth Aurora**, R$ 249,90, 1 un., o
mesmo Endereço de SP → total **R$ 264,90**, centavos `,90`: **recusa** na primeira Tentativa.
Passo 5: **Anotações de um Andarilho**, R$ 39,95 → total **R$ 54,95**, centavos `,95`:
**confirmação que nunca chega**. Nada foi reconfigurado entre A e B — só o Produto mudou.

| # | FR | Dado da demonstração | O que tem de aparecer | Evidência desta execução | Veredito |
|---|---|---|---|---|---|
| 1 | FR-25 | Aurora, total R$ 264,90 | o mesmo checkout, com o total caindo na faixa de recusa | 11:23:55 Carrinho "Subtotal: R$ 249,90"; 11:24:02 Revisão Subtotal R$ 249,90 · Frete (Sudeste) R$ 15,00 · **Total R$ 264,90**; 11:24:16 Confirmar → **AZ-2026-000002** | **aprovado** |
| 2 | FR-27 | — | `PAGAMENTO_RECUSADO` **com o motivo** | 11:24:24, sem recarregar: "Status: **Pagamento recusado**" · **"O Provedor de Pagamento recusou a Tentativa de Pagamento."** · "Restam 2 Tentativas de Pagamento para este Pedido." · "Os Itens de Pedido continuam no próprio Pedido." Histórico: `Aguardando pagamento` 11:24:14 → `Pagamento recusado` 11:24:20, com o motivo na linha | **aprovado** |
| 3 | FR-11, FR-27 | a Página de Produto da Aurora **em outra aba** | o Estoque **voltou** | outra aba, no mesmo instante da reserva ativa (11:24:18): o seletor de quantidade oferecia **9** unidades. Depois da recusa, 11:24:27, a mesma aba recarregada oferecia **≥ 10**. A leitura é pelo seletor, que **satura no teto**: `web/lib/quantidade.ts:5` tem `TETO_POR_ITEM = 10` e a Caixa de compra diz só "Em estoque", sem número, a partir de 10 (`web/app/produtos/[id]/caixa-de-compra.tsx:116`). O teto não enfraquece a conclusão: **9 → ≥ 10** só acontece se a Reserva foi liberada | **aprovado** |
| 4 | FR-27 | "Tentar pagar de novo" | o Pedido seguindo adiante — da segunda Tentativa em diante o Simulado aprova | 11:24:39 nova Tentativa: volta a "Aguardando pagamento" (11:24:37 no histórico) e 11:24:47 "Status: **Pago**" (11:24:44). O histórico guarda as quatro linhas, a recusa incluída — nada foi apagado | **aprovado** |
| 5 | FR-34 | Andarilho, total R$ 54,95, com expiração em 60 s | sair **sozinho** de `AGUARDANDO_PAGAMENTO` por tempo esgotado, e o Estoque voltando | 11:25:47 Confirmar → **AZ-2026-000003**, "Tempo restante para o pagamento: 0:56"; Estoque na outra aba: **9** (mesma leitura pelo seletor do passo 3 — `≥ 10` é o que ele diz no teto). Nenhuma confirmação chegou. 11:26:49, sem recarregar: "Status: Pagamento recusado" · **"Tempo de pagamento expirado."**, histórico `Aguardando pagamento` **11:25:45** → `Pagamento recusado` **11:26:45** — exatamente os **60 s** de `AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO`. Estoque de volta a **≥ 10** às 11:26:51 | **aprovado** |

---

## Roteiro C — operação e reversão (UJ-3, UJ-4)

Dado de demonstração: `admin@azamon.test` em **`/admin/entrar`** (não `/entrar`); Vendedor
"Ensaio 7.4 Distribuidora" e Produto "Luminária de Mesa Ensaio" R$ 129,00 novos; e um Pedido
`PAGO` da Maré para os passos 2 a 4.

**Desvio declarado:** a matriz da spec dizia "o Pedido `PAGO` do Roteiro A". Não dá: o passo 9 do
Roteiro A leva aquele Pedido a `ENTREGUE` de propósito, e em `ENTREGUE` não há transição nem
cancelamento. Cada passo aqui usou um Pedido `PAGO` novo, do mesmo Produto e do mesmo total
(R$ 204,00) — **AZ-2026-000005** no passo 3, **000006** no passo 4 e **000007** na variante *b*.
Na apresentação vale o mesmo: ou se faz o Roteiro C **antes** de o Pedido do Roteiro A passar de
`SEPARANDO`, ou se abre um Pedido novo para ele.

**O Pedido descartado, AZ-2026-000004.** A primeira tentativa dos passos 2 e 3 perdeu a janela de
~30 s: o Pedido nasceu 11:35:29, chegou a `PAGO` 11:35:36, foi a `SEPARANDO` pelo painel 11:35:51
— e a varredura o levou a `ENVIADO` 11:36:21 e `ENTREGUE` 11:36:52 antes de o `Dialog` de
cancelamento ser confirmado. Ele **não sustenta passo nenhum** e foi descartado; o passo 2 e o
passo 3 foram refeitos com AZ-2026-000005. É ele, porém, que explica a aritmética do Estoque da
Maré no resto do Roteiro C: a semente dá 10, o Pedido do Roteiro A consolidou uma unidade em
`ENVIADO` e AZ-2026-000004 consolidou outra (`internal/pedido/maquina.go:132`), então o Estoque
total da Maré era **8** quando o passo 3 começou, às 11:37:26. Com ele, a sequência dos oito
números fecha: 000001 (A), 000002 (B 1–4), 000003 (B 5), **000004 (descartado)**, 000005 (C 3),
000006 (C 4a), 000007 (C 4b) e 000008 (C 5).

**Dois relógios nas células abaixo.** Os horários de *histórico* e de "Última atualização" são do
**servidor**, gravados em `pedido.transicao_status`; os horários de narração ("11:37:57 o painel
respondeu") são do **relógio do condutor do ensaio**, isto é, quando a leitura da tela foi tomada,
sempre igual ou posterior ao do servidor. Onde os dois aparecem na mesma célula, está dito qual é
qual.

| # | FR | Dado da demonstração | O que tem de aparecer | Evidência desta execução | Veredito |
|---|---|---|---|---|---|
| 1 | FR-8, FR-9 | `admin@azamon.test` / `azamon-admin` em `/admin/entrar`; Vendedor e Produto novos | o Produto novo **na Vitrine** | 11:28:24 entrou e caiu em `/admin/vendedores` com os 5 Vendedores da semente. 11:31:40 "Ensaio 7.4 Distribuidora" **Ativo** na tabela. 11:32:10 Produto "Luminária de Mesa Ensaio" R$ 129,00, Estoque 3, Casa e Cozinha, salvo. 11:32:23 na Vitrine, busca `luminaria`: "1–2 de 2 resultados", **Luminária de Mesa Ensaio R$ 129,00 · Em estoque**, com bloco neutro no lugar da imagem (nasceu "Sem imagem") | **aprovado com contorno** (dois perfis de navegador, um por papel — defeito 1) |
| 2 | FR-32 | o Pedido `PAGO` **AZ-2026-000005**, pelo painel | `PAGO` → `SEPARANDO` **pelo painel** | pelo **servidor**: `Pago` **11:37:51**, e `Separando` **11:37:55** — depois da ação, não antes. Pelo **relógio do condutor**: 11:37:54 a tela do Pedido já mostrava `Pago` (a leitura chega até um intervalo de consulta depois do servidor); em `/admin/pedidos` a linha oferecia **"Iniciar a separação"**, o `Dialog` confirmou ("Mudar o Pedido AZ-2026-000005 para Separando? O Pedido não volta ao Status anterior.") e 11:37:57 a leitura seguinte já trazia a resposta do painel, **fora da linha da tabela**: "**Pedido AZ-2026-000005: Separando.**", com a linha passando a oferecer "Registrar o envio". Os 11:37:57 são de quando a tela foi lida; a transição gravada é 11:37:55 | **aprovado com contorno** (dois perfis de navegador — defeito 1) |
| 3 | FR-31 | o mesmo Pedido, agora **como Comprador** | cancelamento pelo Comprador, e o Estoque voltando na Vitrine | 11:38:04, na tela do Pedido: "Cancelar Pedido" → `Dialog` "Cancelar o Pedido AZ-2026-000005? Depois de cancelado, o Pedido não volta a andar." → "Status: **Cancelado**", histórico `Separando` 11:37:55 → `Cancelado` **11:38:02**. Estoque da Maré na Vitrine: **8** antes do Pedido (11:37:26) e **8** depois do cancelamento (11:38:06). **A leitura durante a própria Reserva deste Pedido não foi tomada**: o ensaio mediu antes e depois, e o **7** de 11:39:08 é de outro Pedido (o do passo 4). Então este passo mostra o Estoque **de volta ao valor de antes**, o que é compatível com a Reserva liberada e não a mede; quem mede a liberação na tela é o passo 3 do Roteiro B (9 → ≥ 10, com a leitura tomada **durante** a Reserva) | **aprovado com contorno** (dois perfis de navegador — defeito 1) |
| 4 | FR-31 | um Pedido em `ENVIADO` (**AZ-2026-000006**); e a corrida, com **AZ-2026-000007** | a **recusa** do cancelamento | *(a)* 11:39:12 o painel levou 000006 a `Separando` e depois a **`Enviado`**; na tela do Comprador, 11:39:16: "Status: Enviado" · **"Este Pedido já saiu para entrega e não pode mais ser cancelado."**, e **nenhum** botão de cancelar. *(b)* a recusa **do servidor**: com 000007 em `Separando`, o Comprador abriu o `Dialog` de cancelamento (11:40:10), o Administrador registrou o envio (11:40:13) e o Comprador confirmou — a tela respondeu **"Este Pedido foi enviado enquanto você estava nesta tela e não pode mais ser cancelado."**, com o Status já em `Enviado` e o Pedido intacto | **aprovado com contorno** (dois perfis de navegador — defeito 1) |
| 5 | NFR-7 | Produto novo "Caneca Única do Ensaio", R$ 99,00, **Estoque total 1**; dois Compradores, total R$ 114,00 cada | dois checkouts na última unidade, **um Pedido só** | **condição satisfeita** (abaixo). 11:41:08 o Produto de Estoque 1 na Vitrine; dois Compradores em perfis separados, os dois na Revisão com Total R$ 114,00. 11:43:03 os dois "Confirmar Pedido" disparados em paralelo: o segundo nasceu **AZ-2026-000008** em "Aguardando pagamento"; o primeiro **continuou na Revisão** com **"Estoque insuficiente para este Produto."** e o botão "Voltar ao Carrinho". **Um Pedido, não dois** | **aprovado** |

### A condição do passo 5 do Roteiro C

O passo 5 **só entra na apresentação com o teste da NFR-7 verde** — não se demonstra ao vivo o
que nunca passou automatizado. Rodado no clone limpo, antes do passo, em 2026-09-28:

```
--- PASS: TestSessaoEProduto/a_consistência_de_Estoque_sob_concorrência/a_última_unidade:_oito_checkouts,_um_Pedido (0.35s)
--- PASS: …/três_unidades:_doze_checkouts,_três_Pedidos (0.51s)
--- PASS: …/a_trava_do_Produto_serializa_duas_Reservas_da_última_unidade (0.51s)
ok  github.com/Cashnip/amazon-waddle/api  14.204s
```

O subteste foi rodado com `go test ./api -run TestSessaoEProduto -v` na mesma invocação abaixo, e
`go test ./...` inteiro, no mesmo clone, saiu **tudo `ok`, nenhum `FAIL`**. A invocação, a da
seção Verification da spec, de dentro de `C:\dev\ensaio74` (`go` não roda nesta máquina Windows):

```bash
docker run --rm -v "C:/dev/ensaio74":/src -w //src -v azamon-gocache:/root/.cache \
  -v //var/run/docker.sock:/var/run/docker.sock \
  -e TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal -e TESTCONTAINERS_RYUK_DISABLED=true \
  azamon-dev:latest go test ./...
```

No Git Bash, prefixe com `MSYS_NO_PATHCONV=1`: sem isso o Bash converte `/src` e o `docker run`
morre com "the working directory 'C:/Program Files/Git/src' is invalid". **O passo 5 está
liberado.**
Quem reensaiar rode o teste de novo: verde no dia do ensaio não é verde no dia da apresentação.

---

## O que este ensaio não cobriu

Não ensaiado aqui, e por que:

- **Os passeios pelo teclado.** Este ensaio foi todo por apontador. Os desfechos `,00` e `,90`
  pelo teclado e o painel do Administrador pelo teclado continuam sem registro de quem os viu —
  é a lista do `HANDOFF.md`, e ela não fechou.
- **O `Alert` de preço (D7, FR-19).** Provoca-se mudando o preço de um Produto pelo painel com o
  Carrinho já montado; não é passo de roteiro nenhum do §12, e ficou fora.
- **`AZAMON_ENTREGA_SIMULACAO_ATIVA=false`.** Desligar a simulação é reconfigurar, e as
  Boundaries proíbem reconfigurar durante o ensaio.
- **O Roteiro A com a rede desconectada (NFR-15).** Medido na 1.9 por dentro da rede do compose e
  relatado no `addendum.md` §10; não foi refeito aqui.
- **As quatro larguras das Épicas 3 e 4.** Este ensaio correu numa janela de 1440×900.

Nenhum desses é passo dos três roteiros: a contagem dos 19 continua íntegra.

## Defeitos e riscos que o ensaio achou

Os dois viraram entrada em `deferred-work.md`, com o roteiro e o passo. **Nenhum arquivo de
produção foi alterado por esta estória.**

1. **Um cookie de Sessão por navegador, dois papéis (Roteiro C, passos 1 a 4).** Comprador e
   Administrador compartilham o cookie `azamon_sessao` (`internal/identidade/sessao.go:16`), então
   entrar em `/admin/entrar` **derruba a Sessão do Comprador no mesmo navegador**, sem erro
   nenhum: a próxima visita a `/carrinho` cai calada em `/entrar?destino=%2Fcarrinho`. Foi o que
   aconteceu às 11:32 deste ensaio, e o Roteiro C alterna entre os dois papéis quatro vezes.
   Contornado com **dois perfis de navegador**; quem apresentar precisa do mesmo contorno (ou de
   uma janela privada), senão perde o Carrinho no meio do roteiro.
2. **A janela dos passos 2 e 3 do Roteiro C é de ~30 s (Roteiro C, passos 2 e 3).** Com a
   simulação de entrega ligada — que é o padrão, e é o que faz o passo 9 do Roteiro A andar — a
   varredura move `PAGO` → `SEPARANDO` → `ENVIADO` → `ENTREGUE` de **30 em 30 s**. Quem não mover
   o Pedido pelo painel e cancelá-lo dentro dessa janela encontra o Pedido já `ENVIADO`, e aí o
   passo 3 é impossível e o 4 vira o único disponível. Aconteceu na primeira tentativa deste
   ensaio (11:35:39 `PAGO` → 11:36:52 `ENTREGUE`, com o cancelamento perdido no caminho).

## Registro de ensaios

O §12 pede os três roteiros ensaiados **por ao menos duas pessoas diferentes** antes da
apresentação. Uma delas não pode ser o agente. O formato imita o **Registro de subidas** do
README (§1), e a metade humana fica declaradamente pendente, como a linha da pessoa de fora lá.

| Data | Quem | Commit ensaiado | Máquina · Docker · navegador | Reinício entre roteiros | Resultado (A · B · C) | O que não bateu |
|---|---|---|---|---|---|---|
| 2026-09-28 | **agente** Claude Code, na estória 7.4 — **ensaio técnico, não conta como uma das duas pessoas** | `ddcc361` | Windows 11 Pro · Docker Desktop 29.4.3 · Chrome 153, três perfis · clone em `C:\dev\ensaio74` | **nenhum** — os três roteiros correram sobre o mesmo banco, na sequência, sem `down -v` e sem nada reconfigurado | **9/9 · 5/5 · 5/5** — 19 de 19, nenhum erro visível | os dois itens de "Defeitos e riscos" acima; quatro passos do Roteiro C passaram **com contorno** (dois perfis de navegador) e o passo 3 só na segunda tentativa; o Roteiro C usou Pedidos `PAGO` novos no lugar do Pedido do Roteiro A (desvio declarado na seção do Roteiro C) |
| *em aberto* | **pessoa 1 do time — dono a nomear** | | | | | |
| *em aberto* | **pessoa 2 do time — dono a nomear** | | | | | |

Duas linhas em branco, não uma: o ensaio do agente prova que os roteiros **passam**, não que duas
pessoas do time sabem conduzi-los. A regra do §12 só fecha quando as duas linhas acima tiverem
data, nome e resultado.

## Perguntas prováveis da banca

As seis do §12 mais a do Redis, que o `deck-banca.html` já responde. **A coluna Responsável nasce
em branco de propósito**: os nomes são do time, e o §12 exige responsável nomeado **antes** da
apresentação — ninguém responde por parte que não construiu sem antes ler a fonte.

| Pergunta | Responsável | Onde mora a resposta |
|---|---|---|
| Por que o pagamento é assíncrono, se é simulado? | *a nomear* | PRD §11; `addendum.md` §4; `ARCHITECTURE-SPINE.md` AD-7 e AD-8; seção 06 do `deck-banca.html` |
| Isso é um marketplace ou uma loja? | *a nomear* | PRD §4.2; `addendum.md` §7; AD-2 |
| Como vocês impedem que dois Compradores levem a última unidade? | *a nomear* | NFR-7, FR-24; `addendum.md` §3; AD-4 e AD-5; `api/concorrencia_test.go`; **e o passo 5 do Roteiro C, já ensaiado** |
| Por que não usaram microsserviços? | *a nomear* | PRD §11, inclusive o marco opcional de extração de um serviço; AD-1 e AD-2 |
| E se o pagamento nunca confirmar? | *a nomear* | FR-34; `addendum.md` §2, invariante 6; AD-6; **e o passo 5 do Roteiro B, já ensaiado** |
| Por que não tem Carrinho para visitante? | *a nomear* | `addendum.md` §7; PRD §4.4 |
| Onde exatamente o Redis é usado? | *a nomear* | AD-20; `internal/plataforma/redis.go` é o único cliente — Sessão, token de redefinição e bloqueio de login |

## Armadilhas de quem for apresentar

1. **`docker compose up --build`, nunca `up` sozinho.** `up` não reconstrói: se `azamon` e `web`
   já têm imagem, ela sobe com o código de quando foi construída. Confira a hora em
   `docker compose images` antes de começar — neste ensaio as duas tinham segundos de idade.
2. **Um perfil de navegador por identidade — dois para os passos 1 a 4 do Roteiro C, três se você
   fizer o passo 5.** O cookie de Sessão é o mesmo para Comprador e Administrador: entrar no painel
   numa aba desloga o Comprador na outra, sem aviso. Abra o painel numa janela privada ou num
   perfil separado, e **antes** de montar o Carrinho. O **terceiro** perfil é necessário só no
   passo 5, que precisa de **dois Compradores distintos** disputando a última unidade — este
   ensaio usou `ensaio74@azamon.test` e `ensaio74b@azamon.test`.
3. **O Roteiro C tem 30 s de janela.** Deixe `/admin/pedidos` já aberto e filtrado na outra janela
   **antes** de confirmar o Pedido; do `PAGO` ao `SEPARANDO` pelo painel e daí ao cancelamento são
   duas ações dentro de ~30 s cada. Se perder a janela, use um Pedido novo — não desligue a
   simulação.
4. **O desfecho vem dos centavos do *total*, não de configuração.** Não existe botão para
   forçar recusa. Com Frete de SP (R$ 15,00, inteiro), os centavos do total são os do Produto:
   `,00`–`,89` aprova, `,90`–`,94` recusa, `,95`–`,99` expira. E **a quantidade muda o total**:
   duas unidades do Andarilho somam R$ 79,90 e o total de R$ 94,90 **recusa** em vez de expirar.
5. **`/admin/entrar`, não `/entrar`.** São tabelas separadas e portas separadas; a conta de
   Administrador não entra pela porta do Comprador.
6. **A tabela de Produtos do painel é paginada.** O Produto que você acabou de criar pode não
   estar na página 1 — confira na Vitrine, que é onde a banca vai olhar de todo modo.
7. **Sem `psql`, e sem reinício no meio.** Intervenção no banco reprova o roteiro. Se precisar
   reiniciar, é `docker compose down -v && docker compose up --build`, e aí tudo volta ao estado
   inicial: mesmos 50 Produtos com os mesmos identificadores, nenhum Pedido, e a numeração dos
   Pedidos recomeça em `AZ-2026-000001`. Este ensaio consumiu oito números; um clone novo começa
   do primeiro.
8. **O deck projetado.** O `deck-banca.html` foi emendado nesta estória nos dois pontos em que
   mentia (a porta do pagamento e o schema `pedido`). Se você abrir uma cópia antiga, ela mostra
   `ProvedorDePagamento`, nome que o código não tem.
