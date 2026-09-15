---
title: 'Estória 2.5 — Endereços do Comprador'
type: 'feature'
created: '2026-09-15'
status: 'done'
route: 'dispatch'
baseline_commit: '3c4d6253268a5d45d8b468d383badea78b8accc5'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** `identidade.endereco` não existe. O Comprador não tem onde guardar
para onde a entrega vai, e o checkout da Épica 5 — que congela o Endereço no
Pedido — não tem de onde escolher. A 2.4 deixou o mecanismo de posse provado
com Pedido e prometeu que a 2.5 estende o mesmo teste para Endereço.

**Approach:** O schema `identidade` ganha a tabela; `identidade` ganha a porta
de CRUD com a posse verificada dentro da consulta; `api/` ganha as quatro rotas
de `/api/v1/enderecos`; o `web/` ganha "Meus endereços" com erro em linha e o
estado vazio declarado; e `negacaoPorDono` passa a provar a negação com
Endereço, não só com Pedido.

## Boundaries & Constraints

**Always:** posse verificada na mesma consulta que carrega o dado
(`WHERE id = $1 AND comprador_id = $2`) — AD-11; Endereço de outro dono devolve
o **mesmo** 404 do inexistente; CEP validado no cadastro (formato, PRD §4.2);
erro em linha por campo, com o limite nomeado, ligado por `aria-describedby`
(UX-DR16/UX-DR20c); todo limite de campo na `Config`, com prefixo `AZAMON_` e
linha no `.env`; remover é `DELETE` de verdade — nenhum Pedido é tocado, porque
o Pedido congela o Endereço na criação (AD-3); chave estrangeira cruzando
schema continua proibida (AD-2).

**Never:** coluna de Endereço padrão ou preferido; tela de checkout ou seleção
de Endereço no Pedido (5.2); Menu da conta (2.6); cálculo de Frete e
`pedido.faixa_frete` (5.3); consulta a serviço de CEP pela rede (NFR-15);
regra de negócio no Node (AD-10); erro sentinela novo — `ErrNaoEncontrado` e
`EscreverCampo` já cobrem tudo.

## Decisões

1. **"Escolher" da FR-5 é a listagem, não uma marcação.** `GET /api/v1/enderecos`
   devolve a lista do dono em ordem estável; quem escolhe é o checkout da 5.2,
   sobre essa lista. Uma coluna `padrao` aqui seria estado que nenhuma tela
   desta épica lê e que a 5.2 teria de renegociar.
2. **O CEP é guardado em 8 dígitos**, sem hífen, e a máscara é da tela. Uma
   forma só no banco é o que faz a Faixa de Frete da 5.3 comparar por intervalo
   de texto sem normalizar a cada leitura.
3. **A UF é validada contra as 27 siglas**, e não só por comprimento: `ZZ` passa
   por qualquer teto de dois caracteres e só apareceria como etiqueta impossível
   na entrega.
4. **Os campos são o mínimo de uma etiqueta de entrega** (decisão humana, nem o
   PRD nem o ERD os enumeram): `destinatario`, `cep`, `logradouro`, `numero`,
   `complemento` (o único opcional), `bairro`, `cidade`, `uf`. Sem apelido: a
   lista do checkout distingue um Endereço do outro pelo logradouro, e um rótulo
   livre seria campo que nenhuma FR pede.

## I/O & Edge-Case Matrix

| Cenário | Entrada / Estado | Saída esperada | Erro |
|---|---|---|---|
| Listar sem nenhum | Sessão de A, `GET /api/v1/enderecos` | 200 com lista vazia | — |
| Cadastrar válido | Sessão de A, `POST` com os campos completos | 201 com o Endereço criado | — |
| CEP malformado | `"cep":"1234"` | — | 400 `CAMPO_INVALIDO`, `dados.campo="cep"` |
| UF inexistente | `"uf":"ZZ"` | — | 400 `CAMPO_INVALIDO`, `dados.campo="uf"` |
| Campo acima do teto | `logradouro` com `AZAMON_ENDERECO_TEXTO_MAX`+1 runas | — | 400 nomeando o limite |
| Teto por Comprador | A já tem `AZAMON_ENDERECO_POR_COMPRADOR_MAX` | — | 409 `LIMITE_DE_ENDERECOS` |
| Editar o próprio | Sessão de A, `PUT` sobre Endereço de A | 200 com o Endereço atualizado | — |
| Editar alheio | Sessão de A, `PUT` sobre Endereço de B | resposta **idêntica** à do uuid inexistente | 404 `NAO_ENCONTRADO` |
| Remover o próprio | Sessão de A, `DELETE` sobre Endereço de A | 204, nenhum Pedido alterado | — |
| Remover alheio | Sessão de A, `DELETE` sobre Endereço de B | idêntica à do uuid inexistente | 404 `NAO_ENCONTRADO` |
| Sem Sessão | sem cookie, qualquer rota de Endereço | — | 401 `SESSAO_INVALIDA` |
| Sessão de Administrador | cookie de Administrador em `/api/v1/enderecos` | — | 401 `SESSAO_INVALIDA` |

</frozen-after-approval>

## Code Map

- `db/migracoes/20260911120000_identidade_schema_e_contas.sql` — molde da
  migração nova: `uuidv7()`, `CHECK` na coluna, nenhuma FK cruzando schema. A
  desta estória é um arquivo novo, `*_identidade_endereco.sql`, com
  `+goose Up/Down`; o `sqlc.yaml` recorta o schema de `identidade` por
  `*_identidade_*.sql`, então o prefixo do nome não é decoração.
- `internal/identidade/db/consultas.sql` — `BuscarPedidoDoComprador` do módulo
  `pedido` é a forma do AD-11; aqui nascem `ListarEnderecos`, `CriarEndereco`,
  `AtualizarEndereco` e `RemoverEndereco`, as três últimas com
  `AND comprador_id = @comprador_id` **dentro** da cláusula. **`sqlc` não está
  no PATH:** `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`; o
  gerado é commitado.
- `internal/identidade/identidade.go:59` — `Conta{ID,Nome,Papel}` é o que a
  Sessão carrega; `uuidTexto:141` converte o uuid do pgtype. O Endereço entra
  em arquivo próprio, `internal/identidade/endereco.go`, e só o tipo e as
  quatro funções são exportados — a porta do módulo continua sendo o pacote
  (AD-1).
- `api/pedido.go:110` — `lerPedido` é o molde do handler com dono: resolve
  `compradorDaRequisicao`, chama o módulo, e vira `pgx.ErrNoRows` em
  `erro.ErrNaoEncontrado`. Copiar essa tradução, não inventar outra.
- `api/comprador.go:78` — `validarComprador` é o molde da validação em linha:
  um campo por resposta, runas e não bytes, limiar vindo da `Config`,
  `erro.EscreverCampo`.
- `api/rotas.go:27` — onde as quatro rotas entram, no mux raiz (não no
  administrativo). `decodificarCorpo:99` é o único leitor de corpo JSON e já
  exige `Content-Type: application/json`.
- `internal/plataforma/config.go:83` — onde os dois limiares novos entram.
  `internal/plataforma/env_test.go:19` falha se a chave lida não estiver no
  `.env`, e vice-versa: as duas linhas do `.env` não são opcionais.
- `api/autorizacao_test.go:29` — `negacaoPorDono` prova a igualdade byte a byte
  entre alheio e inexistente. Estender com Endereço (`PUT` e `DELETE`), sem
  reescrever a forma. `api/sessao_test.go:160` — `TestSessaoEProduto` é o único
  `Test…` do pacote; caso novo entra como subteste, no fim, com conta própria.
- `web/app/cadastrar/page.tsx` — molde da tela: `noValidate`, `campoDe()` com
  `aria-describedby` e foco no campo recusado, mensagem sempre a do envelope.
  `web/lib/destino.ts:paraLogin` é o "volta ao ponto em que parou", já pronto.
- **Não tocar:** `internal/pedido/`, `api/admin.go`, `db/semente/`, `media/`.

## Tasks & Acceptance

**Execution:**
- [x] `db/migracoes/20260915120000_identidade_endereco.sql` — tabela
  `identidade.endereco` com `comprador_id uuid NOT NULL REFERENCES identidade.comprador (id)`
  (mesmo schema, então a FK é permitida), `CHECK (cep ~ '^[0-9]{8}$')`,
  `CHECK (uf = upper(uf))` e índice por `comprador_id` — a listagem do dono é a
  única leitura que existe.
- [x] `internal/identidade/db/consultas.sql` — as quatro consultas; rodar
  `sqlc generate`.
- [x] `internal/identidade/endereco.go` — `Endereco`, `ListarEnderecos`,
  `CriarEndereco`, `AtualizarEndereco`, `RemoverEndereco`; a normalização do CEP
  (só dígitos) e da UF (maiúsculas) mora aqui, como `normalizarEmail`, porque a
  coluna tem o `CHECK`.
- [x] `internal/plataforma/config.go` + `.env` — `AZAMON_ENDERECO_TEXTO_MAX`
  (120) e `AZAMON_ENDERECO_POR_COMPRADOR_MAX` (20).
- [x] `api/endereco.go` — os quatro handlers, DTO com os campos decididos, e
  `validarEndereco` no molde de `validarComprador` (CEP, UF, obrigatórios,
  tetos).
- [x] `api/rotas.go` — `GET|POST /api/v1/enderecos` e
  `PUT|DELETE /api/v1/enderecos/{id}`.
- [x] `api/autorizacao_test.go` — `negacaoPorDono` estendida para Endereço.
- [x] `api/endereco_test.go` — a matriz inteira como subtestes de
  `TestSessaoEProduto`, inclusive o teto por Comprador e a prova de que remover
  não altera Pedido.
- [x] `web/app/enderecos/page.tsx` (+ cliente) — lista, formulário de cadastro e
  edição, `Dialog` de confirmação para remover, e o vazio "Nenhum Endereço
  cadastrado." com ação única.
- [x] `README.md` — a rota nova onde as superfícies são listadas.

**Acceptance Criteria:**
- Dado `go test ./...`, quando a suíte roda, então `TestFronteiraDeModulo`,
  `TestDominioNaoConheceHTTP` e os dois testes do `.env` continuam verdes.
- Dado um Comprador com Pedido criado, quando ele remove o Endereço, então o
  Pedido existente continua íntegro — a Épica 5 é que passa a congelar o
  Endereço, e nada nesta estória pode criar dependência que impeça o `DELETE`.
- Dado `cd web && npm run build`, quando o `prebuild` roda, então
  `verificar-offline.mjs` e `node --test` passam — nenhuma fonte ou script de
  rede.

## Implementation Notes

- **O 409 `LIMITE_DE_ENDERECOS` sem sentinela nova.** A matriz pede o código, e
  o "Never" proíbe sentinela nova. Os dois foram honrados por
  `erro.EscreverLimiteDeEnderecos`, no molde exato de `EscreverCampo` e
  `EscreverBloqueio`: não passa pelo `registro` justamente porque a mensagem
  nomeia o limiar, que mora na `Config`.
- **O teto entra no próprio `INSERT`** (`WHERE (SELECT count(*) …) < @maximo`),
  e não num `SELECT count` antes: uma ida ao banco, e zero linhas devolvidas é
  "teto alcançado". `identidade.CriarEndereco` devolve `pgx.ErrNoRows` só nesse
  caso — o dono vem da Sessão, então não há linha alheia nem uuid malformado
  nesse caminho. O teto é READ COMMITTED: dois cadastros simultâneos do mesmo
  Comprador podem passar, com desvio máximo de um Endereço; está escrito na
  função, com o custo de apertar.
- **As duas linhas "alheio" da matriz não estão em `api/endereco_test.go`.**
  Elas são provadas por igualdade byte a byte em `negacaoPorDono`, que foi
  estendida para Endereço por `PUT` e por `DELETE` — uma segunda prova por
  status diria menos. O subteste ainda confere que o Endereço de A sobrevive às
  duas tentativas de B.
- **`db/schema_test.go`** precisou passar de doze para treze tabelas e treze
  chaves primárias. Não estava no Code Map; é o único arquivo tocado fora dele.

## Spec Change Log

## Review Triage Log

### Passo 1 (3 camadas: blind-hunter, edge-case-hunter, verification-gap)

| Veredito | Achado | Evidência |
|---|---|---|
| medium | `remover()` mostra a falha do DELETE e a apaga em seguida | `web/app/enderecos/meus-enderecos.tsx`: o `!resposta.ok` escreve em `erroDaLista`, e o `carregar()` logo abaixo faz `setErroDaLista(null)` no caminho de sucesso. Remover um Endereço que outro dispositivo já apagou fecha o `Dialog` e não diz nada. |
| medium | O `catch` de `remover()` deixa o `Dialog` aberto sobre o alerta | Mesma função: o caminho de rede fora escreve em `erroDaLista` e nunca chama `setARemover(null)`; a mensagem sai atrás do `Dialog`, com o botão reabilitado e sem explicação. |
| medium | Identificador malformado em `PUT`/`DELETE` não tem teste | Pré-verificado pela camada de lacunas: trocar o `pgx.ErrNoRows` de `uuidDe` pelo erro cru do `Scan` derruba o 404 para 500 `INTERNO` e a suíte fica verde. O irmão `GET /pedidos/{id}` já tem o caso (`api/pedido_test.go:293`), e o 500 quebra a própria igualdade do AD-11. |
| medium | `complemento` vazio — o único campo opcional — não tem teste | Pré-verificado: todo corpo válido da suíte carrega `"Apto 12"`. Acrescentar `complemento` aos obrigatórios mantém tudo verde e recusaria a casa sem complemento, que é o Endereço brasileiro mais comum. |
| medium | `Cache-Control: no-store` não é afirmado em nenhuma das quatro rotas | Pré-verificado: o repo afirma o cabeçalho por rota privada (`api/pedido_test.go:48`, `api/redefinicao_test.go:149`) e as de Endereço ficaram de fora. Apagar `semCache` da listagem passa verde — o `cabecalhosComparaveis` do `negacaoPorDono` compara duas respostas do mesmo handler. |
| medium | A ordem estável da listagem é contrato sem asserção | Pré-verificado: tirar o `ORDER BY id` não derruba nada — as três leituras conferem pertinência ou tamanho. A 5.2 herdaria uma lista cuja ordem muda a cada edição, porque o UPDATE reescreve a linha. |
| low | O 401 de `salvar()` e `remover()` retorna sem `setEnviando(false)` | O `return` dentro do `try` pula o `setEnviando(false)` do fim. O `router.push` é navegação de cliente, então a tela fica com todo botão desabilitado enquanto a rota não troca. |
| low | `CHECK (uf = upper(uf))` exige caixa, não forma | `db/migracoes/20260915120000_identidade_endereco.sql`: a coluna aceita `''`, `'SAOPAULO'`, `'1'`. A coluna `cep`, três linhas acima, ganhou `~ '^[0-9]{8}$'` com o argumento de que normalização é restrição de banco — o mesmo argumento vale aqui. |
| low | O CEP sem máscara nunca é exercitado | `formatoCEP` aceita `^[0-9]{5}-?[0-9]{3}$`, e `corpoEnderecoValido` sempre manda `01310-100`. O ramo sem hífen — a forma que o banco guarda, e que um cliente de script mandaria — tem zero cobertura. |
| low | O comentário "a lista reflete" não tem re-listagem depois do `PUT` | `api/endereco_test.go`: a persistência fica provada por acidente da forma do SQL (`RETURNING`), e não pela conferência que o comentário descreve. |
| low | Literais `"20"` e `"120"` onde as constantes existem | `api/endereco_test.go`: o que o subteste prova é que o limiar CONFIGURADO chega à mensagem; escrito como literal, prova que o número 20 chega. |
| low | `map[string]*httptest.ResponseRecorder` na tabela POST/PUT | A ordem de mapa em Go é aleatória: qual dos dois métodos reporta primeiro no `t.Fatalf` muda a cada corrida, sem benefício nenhum. |
| low | `idDe` nasce e `remocaoNaoTocaPedido` o reimplementa em linha | Três funções abaixo da definição, o mesmo status-mais-id escrito à mão. |
| low | O comentário cita `maxLength`, que `campoDe` não emite | `web/app/enderecos/meus-enderecos.tsx`, cabeçalho do componente: deriva de comentário, e o próximo leitor procura o atributo. |
| low | O README documenta as quatro rotas só pela negativa | Elas entram no parágrafo do que a Sessão de Administrador recebe 401, e não onde a superfície da loja é listada; o teto de 20 e o 409 não aparecem em lugar nenhum fora da spec. |
| medium (da triagem) | Nove strings de tela escrevem "endereço" e "pedidos" fora do glossário | Nenhuma das três camadas pegou. O `AGENTS.md` registra que duas revisões já caçaram violações do glossário, e o precedente do repo é literal: `web/app/pedidos/[id]/acompanhamento.tsx:99` escreve "Não foi possível ler o **Pedido**." A tela nova punha "Cadastrar endereço" ao lado do próprio "Nenhum Endereço cadastrado." dela. Corrigido na triagem, junto do 409 de `erro.go` e de uma colisão no README, onde "o endereço se digita" falava da URL dentro da seção de Endereços. O `h1` "Meus endereços" fica como está — é o nome da tela declarado na `EXPERIENCE.md`. |
| low | `sprint-status.yaml` em `in-progress` com a spec em `in-review` | Bookkeeping do próprio workflow — o passo 5 sincroniza. Divergência recorrente deste repo. |
| low (rejeitado) | `criarEndereco` traduz todo `pgx.ErrNoRows` em 409 do teto | Real como armadilha latente, inalcançável hoje: o único outro produtor do sentinela é `uuidDe(compradorID)`, e o identificador vem da Sessão, gravado por `uuidTexto`. Distinguir custaria um sentinela novo e um ramo — complexidade sobre um caminho que nenhum chamador alcança. |
| low (rejeitado) | 200 com corpo ilegível vira "Nenhum Endereço cadastrado." | `corpo ?? []` depois de um `catch(() => null)`. `escreverJSON` sempre emite JSON válido; só um intermediário corrompendo o corpo produz isto, e o conserto acrescenta ramo e mensagem. |
| low (rejeitado) | `identidade.CriarEndereco` não recusa CEP fora de forma antes do CHECK | A porta confia em quem chama, e o único chamador valida. Um guarda ali seria a terceira cópia da mesma regra. |
| low (rejeitado) | Os três ajudantes de requisição de `api/` se sobrepõem | `comCorpo` é superconjunto de `postarComTipo` e `pegarCom`. Real, mas o conserto é refatoração de três arquivos de teste — mais que uma correção direta, sobre duplicação que não muda comportamento. |
| low (rejeitado) | Colunas `text NOT NULL` aceitam `''` e comprimento ilimitado | Endurecimento especulativo: a API já recusa vazio e conta runas, e um CHECK por coluna repetiria a regra num segundo lugar, que divergiria do limiar da Config. |
| low (rejeitado) | Sem `inputMode="numeric"` no campo do CEP | Afordância de teclado em celular, não acessibilidade básica — o campo já tem `autoComplete="postal-code"` e rótulo. |
| low (rejeitado) | O `ON DELETE CASCADE` não tem exercício | Não existe caminho de apagar Comprador no sistema; um teste hoje afirmaria uma garantia sem consumidor. |
| low (rejeitado) | Das 27 siglas só `SP` e `RJ` são aceitas em teste | A própria camada que achou diz o motivo de deixar fora: o único teste barato duplica a constante que ele conferiria. |
| false | O `PUT` alheio poderia sobrescrever e ainda responder 404 | O laço confere o 404 ANTES de qualquer outra coisa (`t.Fatalf`): um PUT que ignorasse a posse devolveria 200 e derrubaria o subteste ali. E a comparação proposta seria cega de qualquer forma — o corpo do PUT é o MESMO `corpoEnderecoValido` com que o Endereço do dono foi criado. |
| false | Oito campos no teto estouram o corpo máximo | `corpoMaximo` é `4 << 10` (`api/sessao.go:17`). Oito campos de 120 runas acentuadas dão ~1.920 bytes, com as chaves ~2,1 KB — metade do teto. |
| false | Nada prova que a listagem é recortada pelo dono | `enderecosDoComprador` exige `[]` literal na conta nova da Nara, e ela roda DEPOIS de `negacaoPorDono`, que deixou um Endereço da Joana gravado. Sem o `WHERE comprador_id`, a lista da Nara traria o da Joana e o subteste cai. |
| false | Nada registra a revisão que levou a 2.4 de `review` a `done` | A 2.4 está em `main` desde `3c4d625`, com a spec em `done` e o log de triagem dela no repositório. O `review` no rastreamento era a divergência recorrente, reconciliada no início desta sessão. |
| false (spec) | `api/endereco_test.go` não tem "a matriz inteira" | O conserto seria editar a tarefa da spec deste build. A auditoria de matriz pede cada linha coberta por ao menos um teste, e as duas linhas "alheio" estão provadas por igualdade byte a byte em `negacaoPorDono`. |
| defer | A casca de cabeçalho está duplicada em seis páginas | Padrão anterior a esta estória (`cadastrar`, `entrar`, `esqueci-a-senha`, `pedidos/[id]`, `produtos/[id]`); esta acrescenta a sexta cópia. A 2.6 edita as seis de qualquer forma, e é lá que a extração se paga. |
| defer | Não há estado de carregamento antes da primeira leitura | Enquanto `enderecos` é `null` a tela mostra só o `h1`. `acompanhamento.tsx` tem o mesmo buraco — é padrão do repo, e consertar só aqui deixaria as duas telas diferentes. |
| defer | `HANDOFF.md` está parado na 2.1 | "Próximo passo: Estória 2.1" com 2.1 a 2.4 em `main`. Anterior a esta estória; reconciliado no fecho desta sessão, junto do `sprint-status.yaml`. |

## Design Notes

**Por que a FK existe aqui.** `endereco.comprador_id` referencia
`identidade.comprador` **dentro do mesmo schema**: o AD-2 proíbe a travessia
entre schemas, não a integridade interna, e é a mesma relação que o ERD desenha
(`COMPRADOR ||--o{ ENDERECO`). É também o que faz o Comprador apagado levar
seus Endereços junto sem código nenhum.

**Por que remover é `DELETE` e não desativação.** O AD-3 já garante o que a FR-5
pede: o Pedido congela o Endereço na criação. Guardar a linha "para o histórico"
duplicaria a garantia e criaria uma lista de Endereços que a tela teria de
filtrar para sempre.

## Verification

**Commands:**
- `go build ./...` e `go test ./...` — verde, com `api/endereco_test.go` novo.
- `cd web && npm run build`.

**Manual checks:**
- `docker compose up` → entrar como Comprador → `/enderecos` mostra "Nenhum
  Endereço cadastrado." → cadastrar → editar → remover.
- `curl -b azamon_sessao=<token de outro Comprador> -X DELETE localhost:8080/api/v1/enderecos/<id alheio>`
  devolve o mesmo envelope 404 de um uuid que nunca existiu.
