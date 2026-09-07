---
name: 'Revisão Adversarial — Espinha de Arquitetura Azamon'
type: review
lens: adversarial-pairwise
target: '_bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md'
created: '2026-09-06'
---

# Revisão Adversarial — Pares de Unidades em Colisão

## Método

Para cada par: duas unidades (das seis funcionalidades do PRD), a escolha concreta que cada uma tomaria lendo só a espinha, a prova de que ambas obedecem a letra de todo AD que poderia impedi-las, o ponto exato onde o sistema quebra quando as duas se encontram, e a Rule executável que fecha o buraco.

Fora de escopo (coberto por `review-concorrencia.md`): entrelaçamento de concorrência, ordem de bloqueio, deadlock, isolamento, idempotência sob corrida, atomicidade de varredura. Os achados abaixo são todos determinísticos — reproduzem em execução single-threaded, sem corrida nenhuma.

Descartados por não fecharem a prova (registrados para não serem redescobertos): "carrinho anônimo antes do login" (o ERD `COMPRADOR ||--|| CARRINHO` e a ausência de FR de guest cart fecham a questão); "quem cria a linha do Carrinho no cadastro" (lazy-creation por `carrinho.ObterOuCriar` é consistente com a direção de dependência do AD-1 e não tem consumidor que exija o contrário); "Endereço congelado vs. FK viva" (a frase "Detalhe do Pedido se ler sem tocar em nenhum outro schema" fecha a leitura); "limite superior de quantidade por item" (não há citação textual que force duas leituras opostas, só meu palpite — descartado por excesso de especulação).

## Sumário

| # | Par | Ponto de quebra | Severidade |
|---|---|---|---|
| 1 | Catálogo × Checkout/Pós-venda | `catalogo.Liberar` trata "sem reserva ativa" como erro em vez de no-op → cancelamento de Pedido em `PAGAMENTO_RECUSADO` falha sempre | CRÍTICO |
| 2 | Catálogo × Checkout | `Reservar`/`Disponivel` ignoram visibilidade (AD-19) → Comprador compra Produto de Vendedor desativado | CRÍTICO |
| 3 | Carrinho × Checkout | Ninguém tem a função para esvaziar o Carrinho na criação do Pedido → recompra do que já foi pago | CRÍTICO |
| 4 | Catálogo × Busca e Navegação | Duas unidades registram o handler de `GET /api/v1/produtos` | CRÍTICO |
| 5 | Carrinho × Checkout | `preco_visto_centavos`: quem detecta e "consome" a mudança de preço ("de X para Y") | ALTO |
| 6 | Checkout × Pós-venda | "Estado terminal" do Pedido — duas definições dentro do mesmo pacote `internal/pedido` | MÉDIO |

---

## Par 1 — Catálogo × Checkout/Pós-venda: `catalogo.Liberar` sem reserva ativa

**CRÍTICO**

### As duas unidades

- **Catálogo** — dona de `catalogo.Liberar`, chamada pela tabela do AD-3.
- **Checkout/Pós-venda** — dona de `pedido.Transicionar`, que executa o efeito da terceira coluna do AD-3, incluindo a linha de cancelamento (FR-31, funcionalidade Pedidos e Pós-venda) e a linha de recusa (FR-27, funcionalidade Checkout). As duas linhas chamam a mesma função `Liberar`.

### Escolha concreta de cada uma

A Rule do AD-3 lista, para cancelamento: `{AGUARDANDO_PAGAMENTO, PAGAMENTO_RECUSADO, PAGO, SEPARANDO} → CANCELADO | Comprador (FR-31) | catalogo.Liberar, se ativa`. `PAGAMENTO_RECUSADO` está nessa lista de origens — mas a própria tabela já mandou liberar a reserva quando o Pedido **entrou** em `PAGAMENTO_RECUSADO` (linha anterior: `AGUARDANDO_PAGAMENTO → PAGAMENTO_RECUSADO | catalogo.Liberar`). Ou seja, cancelar a partir de `PAGAMENTO_RECUSADO` é o único ramo em que "se ativa" é sempre falso.

**Catálogo implementa** (lendo só AD-5, que nunca menciona o caso "sem reserva", e AD-14, que proíbe engolir erro):

```go
// internal/catalogo/servico.go
func (s *Servico) Liberar(ctx context.Context, tx pgx.Tx, pedidoID uuid.UUID) error {
    tag, err := tx.Exec(ctx, `
        UPDATE catalogo.reserva_estoque
        SET status = 'liberada' WHERE pedido_id = $1 AND status = 'ativa'`, pedidoID)
    if err != nil { return err }
    if tag.RowsAffected() == 0 {
        return catalogo.ErrReservaInexistente // "ou é tratado, ou sobe" — AD-14
    }
    return nil
}
```

**Checkout/Pós-venda implementa** (lendo só AD-3, cuja tabela é "exaustiva e obrigatória", e AD-4, que só exige que o efeito rode na mesma transação — nada ali pede uma checagem prévia):

```go
// internal/pedido/servico.go
func (s *Servico) Transicionar(ctx context.Context, tx pgx.Tx, pedidoID uuid.UUID, esperado, novo Status, ator Ator, motivo *string) error {
    // ... valida esperado == atual, grava transicao_status ...
    if novo == CANCELADO {
        if err := s.catalogo.Liberar(ctx, tx, pedidoID); err != nil {
            return err // efeito obrigatório da terceira coluna, AD-3/AD-4
        }
    }
    return nil
}
```

### Prova de conformidade

- **AD-3** só impede que exista uma transição fora da tabela, ou que falte o efeito da terceira coluna. Chamar `Liberar` incondicionalmente **é** cumprir a Rule ao pé da letra — a tabela não distingue "chame só se ativa" de "chame sempre e deixe `Liberar` decidir". A cláusula "se ativa" fica do lado de fora da assinatura de `Transicionar`, sem dono declarado.
- **AD-5** define `Reservar`, `Liberar`, `Consolidar` só em termos de efeito sobre estoque quando há o que fazer; nunca fala do caso vazio. Retornar erro no caso vazio não viola nenhuma frase do AD-5.
- **AD-14** ("Erro engolido é proibido: ou é tratado, ou sobe") **empurra** o Catálogo exatamente para a implementação que quebra: se ele fizesse *no-op* silencioso sem sinalizar nada, isso poderia ser lido como "engolir" uma situação anômala. Falhar é a leitura mais defensável do próprio AD-14.
- **AD-4** exige que o efeito rode na mesma transação do `Transicionar` — obedecido nos dois lados.

Nenhum AD nomeia "idempotência de `Liberar`" nem diz que "se ativa" é uma checagem que `pedido` deve fazer antes de chamar. As duas implementações são, cada uma isoladamente, a leitura mais natural do seu próprio AD.

### Onde quebra

Roteiro: Comprador finaliza o checkout (reserva criada) → Confirmação de recusa chega (FR-27): `AGUARDANDO_PAGAMENTO → PAGAMENTO_RECUSADO`, `Liberar` roda, reserva marcada `liberada`. Comprador desiste e cancela o Pedido (FR-31): `PAGAMENTO_RECUSADO → CANCELADO`. `Transicionar` chama `Liberar` de novo → `UPDATE ... WHERE status = 'ativa'` afeta zero linhas → `ErrReservaInexistente` sobe → a transação inteira do cancelamento sofre rollback (AD-4: efeito e transição vivem na mesma transação — se o efeito falha, a transição não é gravada). **O Comprador nunca consegue cancelar um Pedido em `PAGAMENTO_RECUSADO`**, um dos quatro estados de origem que a própria tabela do AD-3 declara válidos para FR-31. Como este é exatamente um dos três ramos de recusa que a espinha manda tratar "nunca uma só" (seção "Três recusas distintas"), é also o caminho mais provável de ser exercitado na demonstração.

Cada lado, testado isoladamente (máquina de estados em memória, mockando o outro módulo — convenção de Testes), passa: o teste de Catálogo verifica que `Liberar` falha quando não há reserva (comportamento correto do ponto de vista dele); o teste de Checkout verifica que `Transicionar` chama `Liberar` e propaga erro (comportamento correto do ponto de vista dele). Só a integração revela o buraco.

### Rule que fecha o buraco

> **AD-5, aperto.** `catalogo.Liberar(ctx, tx, pedidoID) error` é idempotente: se não existe reserva ativa para `pedidoID`, retorna `nil`, não erro. O "se ativa" da tabela do AD-3 é uma nota sobre o efeito observável, nunca uma checagem que `pedido` deve fazer antes de chamar — `pedido` chama `Liberar` incondicionalmente em toda transição para `CANCELADO` e em toda recusa. A mesma regra vale para `Consolidar` (baixar o total de uma reserva já baixada é no-op) e para `Reservar` com lista de itens vazia (no-op, nunca erro). Toda função de mutação de Estoque documenta, na própria assinatura, seu comportamento no caso "já feito" ou "nada a fazer" — silêncio sobre isso na Rule do AD-5 é o próprio defeito.

---

## Par 2 — Catálogo × Checkout: `Reservar`/`Disponivel` e a visibilidade

**CRÍTICO**

### As duas unidades

- **Catálogo** — dona de `catalogo.Reservar` e `catalogo.Disponivel` (AD-5), e do predicado `catalogo.Visiveis` (AD-19).
- **Checkout** — dona da criação do Pedido (`pedido.Criar`, linha "criação → `AGUARDANDO_PAGAMENTO`" do AD-3), que chama `Reservar`.

### Escolha concreta de cada uma

O "Prevents" do AD-19 nomeia literalmente o cenário que este par produz: *"a Vitrine mostrar um Produto que a busca esconde, ou **o Carrinho aceitar um Produto de Vendedor desativado**"*. A Rule do AD-19, no entanto, só amarra `Visiveis` a leitura: *"Vitrine, página de Produto, Carrinho e criação de Pedido passam todos por ele"* — sem dizer que `Reservar` (a escrita que efetivamente comete o Comprador a comprar) também aplica esse filtro.

**Catálogo implementa** `Reservar` lendo só o AD-5, que fala exclusivamente de estoque:

```go
// internal/catalogo/servico.go
func (s *Servico) Reservar(ctx context.Context, tx pgx.Tx, itens []ItemParaReservar) error {
    for _, item := range itens {
        var estoqueTotal, reservadoAtivo int
        row := tx.QueryRow(ctx, `SELECT estoque_total FROM catalogo.produto WHERE id = $1 FOR UPDATE`, item.ProdutoID)
        // ... soma reservas ativas, compara com quantidade pedida ...
        if disponivel := estoqueTotal - reservadoAtivo; disponivel < item.Quantidade {
            return catalogo.ErrEstoqueInsuficiente // tudo-ou-nada, FR-24
        }
    }
    // nenhuma linha checa produto.ativo nem vendedor.ativo — Reservar é sobre estoque, a Rule do AD-5 não pede mais
    // ... insere as linhas em reserva_estoque ...
    return nil
}

func (s *Servico) Disponivel(ctx context.Context, tx pgx.Tx, produtoIDs []uuid.UUID) map[uuid.UUID]int {
    // idem: estoque_total − Σ reservas ativas, sem join com vendedor nem checagem de produto.ativo
}
```

**Checkout implementa** `pedido.Criar` lendo só a tabela do AD-3, que declara exaustivamente que o único efeito obrigatório da criação é `catalogo.Reservar`:

```go
// internal/pedido/servico.go
func (s *Servico) Criar(ctx context.Context, tx pgx.Tx, compradorID uuid.UUID, itens []ItemCarrinho, enderecoID uuid.UUID) (Pedido, error) {
    if err := s.catalogo.Reservar(ctx, tx, converter(itens)); err != nil {
        return Pedido{}, err // efeito obrigatório da linha 1 da tabela do AD-3 — a tabela não lista mais nada
    }
    // grava pedido, item_pedido (congela nome, preço, Vendedor), transicao_status
}
```

### Prova de conformidade

- **AD-5** define `Reservar` e `Disponivel` só em termos de `estoque_total − Σ reservas ativas`. Nenhuma frase menciona `ativo` ou `vendedor_ativo`. Implementar exatamente essa fórmula, nem mais nem menos, é obedecer à letra.
- **AD-3** declara a tabela "exaustiva e obrigatória" para o que a criação do Pedido faz — o efeito listado é `catalogo.Reservar`, ponto. Um dev que adicionasse uma chamada extra a `Visiveis` estaria acrescentando algo que a própria Rule não pede (e que o espírito ponytail do documento — "cerimônia, não arquitetura" — desencoraja).
- **AD-19** só cita "criação de Pedido" como algo que "passa por" `Visiveis` no sentido de leitura de listagem (Vitrine, página de Produto) — a frase não distingue "consultar para listar" de "consultar para autorizar a escrita", e o Checkout, focado na tabela do AD-3, nunca precisa abrir o AD-19 para construir `Criar`.
- **AD-11** (autorização por dono do recurso) não se aplica — visibilidade de Produto não é posse de Comprador.

Nenhum AD amarra `Reservar` ou `Disponivel` ao predicado de visibilidade de forma que force uma leitura só.

### Onde quebra

Administrador desativa um Vendedor (ação de manutenção do Catálogo) enquanto um Comprador já tem, no carrinho, um Produto desse Vendedor com estoque físico positivo. O Comprador finaliza o checkout: `Disponivel` (chamado para montar a tela, AD-18) devolve a contagem física, sem saber que o Vendedor sumiu; `Reservar` sucede pelo mesmo motivo. O Pedido é criado, pago e processado normalmente — o Comprador comprou de um Vendedor desativado, exatamente o cenário que o "Prevents" do AD-19 promete que não acontece. A garantia existe em prosa, não na função que efetivamente commita a venda.

### Rule que fecha o buraco

> **AD-19, aperto.** `catalogo.Disponivel` retorna `0` (nunca omite a chave do mapa, nunca lança erro) para Produto inexistente ou invisível — "disponível" já embute "comprável". `catalogo.Reservar` verifica visibilidade **na mesma consulta `FOR UPDATE`** que verifica estoque: reservar um Produto invisível falha com o mesmo `catalogo.ErrEstoqueInsuficiente` que estoque zerado (do ponto de vista do Comprador é a mesma frase: "não posso comprar isso agora"). A lista de "quem passa pelo predicado" do AD-19 inclui, a partir de agora, as próprias funções de escrita do AD-5 — `Visiveis` não é só para telas de listagem.

---

## Par 3 — Carrinho × Checkout: quem esvazia o Carrinho na criação do Pedido

**CRÍTICO**

### As duas unidades

- **Carrinho** — dona do schema `carrinho` e de `internal/carrinho/carrinho.go`, a única porta de entrada para mutar o Carrinho (regra de módulo do Paradigma de Design).
- **Checkout** — dona de `pedido.Criar`, que consome os itens do Carrinho para montar o Pedido.

### Escolha concreta de cada uma

A tabela do AD-3, "exaustiva e obrigatória", lista para a criação do Pedido só um efeito: `catalogo.Reservar`. **Nenhuma linha da tabela, nenhuma frase do documento, menciona esvaziar o Carrinho** (busquei "esvazia" no texto inteiro — não ocorre). Mas o diagrama do AD-1 é claro sobre a direção: existe a seta `pedido --> carrinho`; **não existe** `carrinho --> pedido` — e "uma seta que não está aqui é defeito, uma seta invertida é ciclo". Logo, o Carrinho não pode reagir a um Pedido criado; só o Pedido pode agir sobre o Carrinho.

**Carrinho implementa** `carrinho.go` com o que a Rule dele (AD-5, AD-9, AD-11, AD-17, AD-19) pede — nada ali fala de ser mutado por outro módulo:

```go
// internal/carrinho/carrinho.go — único arquivo público
func Adicionar(ctx context.Context, tx pgx.Tx, compradorID, produtoID uuid.UUID, qtd int) error
func RemoverItem(ctx context.Context, tx pgx.Tx, itemID uuid.UUID) error
func AlterarQuantidade(ctx context.Context, tx pgx.Tx, itemID uuid.UUID, qtd int) error
func Itens(ctx context.Context, compradorID uuid.UUID) ([]ItemCarrinho, error) // leitura
// sem função de escrita em lote — cada endpoint HTTP de carrinho já cobre adicionar/remover/alterar um item por vez
```

Do lado do front-end (`web/`), a interpretação simétrica: depois que `POST /api/v1/pedidos` responde `201`, é o **navegador** quem dispara `DELETE /api/v1/carrinho` para limpar a tela — parece natural porque cada endpoint em `api/` já corresponde a "uma ação da tela" (Paradigma de Design: "Tradução... Nenhuma regra de domínio", o que sugere handlers finos e independentes, não uma orquestração cross-domínio escondida em um handler só).

**Checkout implementa** `pedido.Criar` esperando que o esvaziamento seja parte do mesmo caso de uso, porque o AD-4 é categórico: *"quem inicia o caso de uso abre a transação e a passa adiante"* — e criar o Pedido e esvaziar o Carrinho de origem são, para quem lê o AD-4, obviamente **um** caso de uso, não dois:

```go
// internal/pedido/servico.go
func (s *Servico) Criar(ctx context.Context, tx pgx.Tx, compradorID uuid.UUID, enderecoID uuid.UUID) (Pedido, error) {
    itens, _ := s.carrinho.Itens(ctx, compradorID)
    if err := s.catalogo.Reservar(ctx, tx, converter(itens)); err != nil { return Pedido{}, err }
    // ... grava pedido, item_pedido ...
    if err := s.carrinho.Esvaziar(ctx, tx, compradorID); err != nil { return Pedido{}, err } // <- não existe em carrinho.go
    return pedidoCriado, nil
}
```

### Prova de conformidade

- **AD-3**: a tabela é exaustiva sobre o efeito **sobre o Estoque** — título da terceira coluna é literalmente "Efeito obrigatório sobre o Estoque". Esvaziar o Carrinho não é um efeito sobre o Estoque, então a tabela simplesmente não fala sobre isso — não é uma proibição, é uma lacuna de escopo. Ambos os lados podem honestamente dizer "a tabela não me contradiz".
- **AD-4**: exige transação explícita passada como parâmetro — obedecido pela versão do Checkout (que chama tudo com o mesmo `tx`) e irrelevante para a versão do Carrinho (que não sabe que precisa participar).
- **AD-1**: a seta `pedido --> carrinho` permite exatamente a chamada que o Checkout tenta fazer — o AD-1 não impede a existência de `Esvaziar`, só não a exige.
- **Paradigma de Design** ("um módulo só é alcançado pela sua interface pública"): o Carrinho respeitou isso ao construir `carrinho.go` só com o que a *sua própria* lista de ADs (AD-5, AD-9, AD-11, AD-17, AD-19) pede.

Nenhum AD nomeia "esvaziar o Carrinho" como responsabilidade de ninguém.

### Onde quebra

Duas variantes, ambas reais:

1. **Falha de build.** `pedido.Criar` chama `s.carrinho.Esvaziar(...)`, que não existe em `carrinho.go`. Erro de compilação na integração dos dois PRs — barato de achar, caro de decidir quem "cede" (o Carrinho ganha uma função que nunca imaginou expor; o Checkout descobre que precisa saber o `tx` de dentro do Carrinho, quebrando a suposição de que "cada endpoint é uma ação simples").
2. **Pior: os dois lados compilam, e o Carrinho nunca é esvaziado.** Se, para evitar o acoplamento, a equipe resolve que o **navegador** dispara `DELETE /api/v1/carrinho` depois de `POST /api/v1/pedidos` ter sucesso — duas requisições, duas transações — existe uma janela onde a primeira teve sucesso e a segunda nunca chega (aba fechada, rede caiu, o dev de tela do Carrinho nunca implementou essa chamada porque, do lado dele, "isso é responsabilidade do Checkout"). O Comprador vê, depois de pagar, o carrinho **ainda cheio** com o produto que acabou de comprar — pode tentar comprar de novo o mesmo item, cuja unidade de estoque já está presa à Reserva do primeiro Pedido. É o tipo de bug que só aparece ao vivo, na demonstração, exatamente onde o NFR-15 já exige robustez sem rede.

### Rule que fecha o buraco

> **AD-3, aperto.** A linha "criação → `AGUARDANDO_PAGAMENTO`" da tabela ganha um segundo efeito obrigatório, na mesma transação: `carrinho.Esvaziar(ctx, tx, compradorID, itemIDs)`, chamado por `pedido.Criar` depois de `catalogo.Reservar` suceder. `carrinho.go` declara essa função na interface pública — é o único ponto de escrita do Carrinho que outro módulo pode chamar, e só `pedido` o chama, só durante a criação. Nenhuma chamada HTTP separada do navegador esvazia o Carrinho: não existe `DELETE /api/v1/carrinho` para esse fim. Título da terceira coluna do AD-3 passa a ser "Efeito obrigatório sobre Estoque e Carrinho".

---

## Par 4 — Catálogo × Busca e Navegação: duas donas para `GET /api/v1/produtos`

**CRÍTICO**

### As duas unidades

- **Catálogo** (FR-6..FR-11), governada por AD-2, AD-5, AD-9, AD-12, **AD-19**.
- **Busca e Navegação** (FR-12..FR-15), governada por **AD-16**, AD-2, AD-19.

### Escolha concreta de cada uma

A convenção "Estado de listagem" fixa uma única forma de URL para toda listagem de Produto: `?termo=&categoria=&pagina=` — os três parâmetros juntos, sem distinguir rota. A convenção de Rotas fixa `/api/v1/produtos` como o recurso plural. Nada no documento diz explicitamente qual módulo registra o handler HTTP dessa rota quando `termo` está vazio (navegação pura por categoria — a "Vitrine").

**Dev de Catálogo**, lendo o conjunto de ADs que o Mapa Funcionalidade → Arquitetura atribui à sua feature, encontra o AD-19: *"Vitrine, página de Produto, Carrinho e criação de Pedido passam todos por [`catalogo.Visiveis`]"* — a Vitrine está listada ao lado da "página de Produto", que é inequivocamente dele. Ele implementa:

```go
// api/produtos.go
mux.HandleFunc("GET /api/v1/produtos", func(w http.ResponseWriter, r *http.Request) {
    categoria := r.URL.Query().Get("categoria")
    pagina, _ := strconv.Atoi(r.URL.Query().Get("pagina"))
    itens, total := catalogoSvc.ListarVisiveis(r.Context(), categoria, pagina, tamanhoPagina)
    responderJSON(w, ListaProdutosDTO{Itens: itens, Total: total})
})
```

**Dev de Busca e Navegação**, lendo o conjunto de ADs da sua própria feature, encontra o AD-16: *"nenhum módulo além de `busca` monta consulta para localizar Produto"* — e o próprio nome da feature é "Busca **e Navegação**". Ele lê "navegação por categoria, sem termo" como parte do que já é dele por nome, e implementa:

```go
// api/busca.go
mux.HandleFunc("GET /api/v1/produtos", func(w http.ResponseWriter, r *http.Request) {
    termo := r.URL.Query().Get("termo") // pode vir vazio — "localizar sem termo" ainda é localizar
    categoria := r.URL.Query().Get("categoria")
    pagina, _ := strconv.Atoi(r.URL.Query().Get("pagina"))
    itens, total := buscaSvc.Localizar(r.Context(), termo, categoria, pagina, tamanhoPagina)
    responderJSON(w, ListaProdutosDTO{Itens: itens, Total: total})
})
```

### Prova de conformidade

- **AD-16** proíbe *outro módulo* montar consulta para localizar Produto — não proíbe Catálogo de **listar os que ele mesmo possui**, uma leitura que o dev de Catálogo sustenta dizendo que "listar por categoria" não é "localizar" (localizar = busca textual/relevância). AD-16 não define o verbo "localizar" com precisão suficiente para fechar essa leitura.
- **AD-19** cita "Vitrine" ao lado de "página de Produto" como consumidoras de `catalogo.Visiveis` — o dev de Catálogo não está inventando essa leitura, está citando o texto.
- **AD-2**: ambas as implementações respeitam "uma consulta lê apenas o próprio schema" — Catálogo lê `catalogo`, Busca lê a VIEW dedicada.
- **AD-1**: `api --> catalogo` e `api --> busca` são setas válidas e ambas existem no diagrama; nada ali arbitra qual delas serve qual rota.
- A convenção de Rotas fixa o **caminho** (`/api/v1/produtos`) mas não amarra caminho a módulo — só a tabela do Mapa Funcionalidade → Arquitetura faz esse tipo de associação, e ela associa rota a *feature*, nunca a *handler*.

Nenhum AD nomeia o dono do endpoint HTTP; a Rule mais próxima (AD-16) é lida por cada lado a seu favor porque o verbo-chave ("localizar") não está definido.

### Onde quebra

`net/http.ServeMux` (Go stdlib, a Rule do documento fixa "Roteamento HTTP: `net/http.ServeMux` da biblioteca padrão") entra em pânico ao registrar dois handlers para o **mesmo padrão** de rota (`"GET /api/v1/produtos"`) no arranque do binário — `docker compose up` nunca sobe, e o erro só aparece quando os dois PRs (Catálogo e Busca) são integrados no mesmo binário, não durante o desenvolvimento isolado de cada um (cada dev roda só seu handler localmente). Mesmo se, por sorte de nomenclatura, cada um escolher um caminho ligeiramente diferente (`/produtos` vs. `/busca`) e evitar o panic, a convenção "Estado de listagem... o link reproduz a tela" quebra: a Vitrine (sem termo) e a Busca (com termo) passam a ser páginas com URLs e formatos de resposta potencialmente diferentes, e a tela de front-end não tem como saber, a partir da própria espinha, qual endpoint chamar quando o Comprador limpa o campo de busca mas mantém um filtro de categoria.

### Rule que fecha o buraco

> **AD-16, aperto.** `GET /api/v1/produtos` é servido exclusivamente por `busca`, com ou sem `termo` — "listar sem termo" é "buscar com termo vazio", nunca uma rota separada. `catalogo` nunca registra handler HTTP para listar ou paginar Produto; expõe só `ObterVisivel(id)` (lookup direto, página de Produto) e a função `Visiveis`/`Disponivel` para outros módulos consumirem via chamada Go. Tabela de posse de rota, exaustiva como a do AD-1: toda rota de leitura de Produto (`/api/v1/produtos`, `/api/v1/produtos/{id}`) tem exatamente uma linha aqui, e uma rota sem linha é defeito.

---

## Par 5 — Carrinho × Checkout: quem "resolve" a mudança de preço

**ALTO**

### As duas unidades

- **Carrinho** — dona de `item_carrinho.preco_visto_centavos` (schema `carrinho`, AD-2).
- **Checkout** — dona da "entrada do checkout", onde o AD-17 localiza o recálculo da FR-19.

### Escolha concreta de cada uma

A tabela de referências cruzadas descreve o que o Carrinho congela: *"`preco_visto_centavos` — o preço no momento da última alteração, para a FR-19 dizer 'de X para Y'"*. O AD-17 diz: *"O Frete é congelado no Pedido na criação, e o recálculo da FR-19 acontece na entrada do checkout, que é `pedido`."* As duas frases atribuem a mesma responsabilidade (mostrar "de X para Y") a dois lugares diferentes, sem dizer qual escreve o novo valor depois de mostrado.

**Carrinho implementa**, lendo só a tabela de referências cruzadas (que é a única frase que menciona `preco_visto_centavos`), a atualização na própria leitura do carrinho — é o único evento que o módulo Carrinho enxerga:

```go
// internal/carrinho/servico.go
func (s *Servico) ObterCarrinho(ctx context.Context, compradorID uuid.UUID) (CarrinhoDTO, error) {
    itens, _ := s.repo.Itens(ctx, compradorID)
    precos := s.catalogo.PrecosAtuais(ctx, idsDe(itens)) // chamada pública a catalogo
    for _, item := range itens {
        if atual := precos[item.ProdutoID]; atual != item.PrecoVistoCentavos {
            item.PrecoAnteriorCentavos = &item.PrecoVistoCentavos // "de X"
            item.PrecoAtualCentavos = atual                        // "para Y"
            s.repo.AtualizarPrecoVisto(ctx, item.ID, atual)        // já resolve — próxima leitura não mostra mais
        }
    }
    return montarDTO(itens), nil
}
```

**Checkout implementa**, lendo só o AD-17 (que é a Rule sob a qual a feature Checkout vive, conforme o Mapa Funcionalidade → Arquitetura), a comparação na entrada do checkout, usando a seta `pedido --> carrinho` do AD-1:

```go
// internal/pedido/servico.go
func (s *Servico) EntrarNoCheckout(ctx context.Context, compradorID uuid.UUID) (RevisaoDTO, error) {
    itens, _ := s.carrinho.Itens(ctx, compradorID) // função pública de leitura do carrinho
    precos := s.catalogo.PrecosAtuais(ctx, idsDe(itens))
    var revisao RevisaoDTO
    for _, item := range itens {
        if atual := precos[item.ProdutoID]; atual != item.PrecoVistoCentavos {
            revisao.Alteracoes = append(revisao.Alteracoes,
                fmt.Sprintf("de %d para %d", item.PrecoVistoCentavos, atual))
        }
    }
    return revisao, nil
}
```

### Prova de conformidade

- **AD-2**: cada função só lê/escreve o próprio schema, e consome o outro módulo por chamada Go, não SQL cruzado — obedecido nos dois lados.
- **AD-17**: a frase "o recálculo da FR-19 acontece na entrada do checkout, que é `pedido`" é satisfeita literalmente pela implementação do Checkout — ele não precisa saber que o Carrinho já faz algo parecido, porque o AD-17 não menciona o Carrinho nesse ponto (o AD-17 fala do Carrinho só para dizer que ele **não calcula Frete**, um assunto diferente).
- **AD-1**: `pedido --> carrinho` é seta válida; `carrinho.Itens` é leitura pública, permitida.
- **AD-9**: ambos em `int64`/centavos.

Nenhum AD diz que `carrinho.Itens` deve ser (ou não ser) livre de efeito colateral, nem que só um dos dois lugares pode escrever em `preco_visto_centavos`.

### Onde quebra

O Comprador visita a tela do Carrinho (dispara `ObterCarrinho`, que já compara e **atualiza** `preco_visto_centavos` para o preço atual) e, na sequência, entra no checkout. `EntrarNoCheckout` chama `carrinho.Itens`, que devolve o preço **já igualado** ao atual pela visita anterior — a comparação de `pedido` nunca encontra diferença, e a "revisão do checkout" nunca exibe "de X para Y", mesmo que o preço tenha de fato mudado entre a adição do item e o pagamento (a mudança só não sobrevive entre a tela do Carrinho e a tela de Checkout, que é exatamente onde a Rule do AD-17 diz que ela deveria aparecer). Se, em vez disso, a mesma função de leitura de baixo nível for reaproveitada por ambos os call-sites (API HTTP do Carrinho e a chamada feita por `pedido`), o problema se agrava: a própria chamada que `pedido` faz para **ler** já dispara, como efeito colateral dentro do módulo Carrinho, a escrita que apaga a diferença — o checkout está comparando um valor com ele mesmo.

### Rule que fecha o buraco

> **AD-17, aperto.** `carrinho.Itens(ctx, compradorID)` é leitura pura — nunca escreve `preco_visto_centavos`, em nenhum call-site. Só uma função, `carrinho.ConfirmarPrecoVisto(ctx, tx, itemID, precoAtual)`, atualiza esse campo, e só `pedido` a chama, dentro da mesma transação da entrada no checkout (AD-4), depois de já ter lido e reportado a diferença na `RevisaoDTO`. A tela do Carrinho mostra o preço atual ao lado do preço visto (dado read-only, calculado a cada leitura, nunca persistido) sem nunca gravar nada — quem persiste a "ciência" da mudança é sempre o Checkout, uma vez, no momento em que o Comprador de fato avança.

---

## Par 6 — Checkout × Pós-venda: o que é "estado terminal" do Pedido

**MÉDIO**

### As duas unidades

- **Checkout** (AD-3, AD-4, AD-5, AD-6, AD-7, AD-8, AD-9, AD-17, **AD-18**) — implementa o polling de `GET /pedidos/<id>` a cada 3s.
- **Pós-venda** (AD-3, AD-4, AD-6, AD-15, **AD-18**) — implementa o polling de `GET /pedidos/<id>` a cada 10s e o de `GET /admin/pedidos` a cada 10s "enquanto listar Pedido não terminal".

Ambas vivem em `internal/pedido` (mesmo pacote Go), mas são duas linhas separadas do Mapa Funcionalidade → Arquitetura, escritas por dois estudantes.

### Escolha concreta de cada uma

O AD-18 usa "terminal" duas vezes sem jamais listar quais estados são terminais. Isso é dedutível da tabela do AD-3 (só `ENTREGUE` e `CANCELADO` nunca aparecem como origem de uma transição) — mas é dedução, não citação.

**Dev de Checkout**, focado no seu próprio recorte do fluxo (que só se importa em saber quando parar de reconsultar a cada 3s durante o pagamento), escreve:

```go
// internal/pedido/pedido.go
func EhTerminalParaPagamento(s Status) bool {
    return s != AGUARDANDO_PAGAMENTO && s != PAGAMENTO_RECUSADO // "saiu do fluxo de pagamento"
}
```

**Dev de Pós-venda**, implementando o polling de 10s do painel do Administrador (que precisa parar só quando não há mais nada a acontecer com o Pedido), escreve uma função com o mesmo nome óbvio, sem saber que Checkout já criou uma:

```go
// internal/pedido/pedido.go
func EhTerminal(s Status) bool {
    return s == ENTREGUE || s == CANCELADO
}
```

### Prova de conformidade

- **AD-18** exige que exista alguma noção de "terminal" para as três superfícies de polling que ele mesmo declara exaustivas — não define o conjunto, então qualquer conjunto dedutível da tabela do AD-3 é defensável.
- **AD-3**: a tabela, lida por quem só olha o recorte do pagamento (`AGUARDANDO_PAGAMENTO`/`PAGAMENTO_RECUSADO`/`PAGO`), sustenta a leitura de Checkout — ele para de reconsultar assim que sai desses três; não precisa saber que `PAGO` ainda vai virar `SEPARANDO`/`ENVIADO`/`ENTREGUE`, porque isso é Pós-venda, uma feature diferente que ele não leu a fundo (cada estudante lê a espinha inteira, mas otimiza para os ADs que o Mapa atribui à própria feature).
- **AD-4/AD-6**: irrelevantes para este ponto — nenhuma mutação aqui, só leitura de status para decidir polling.

### Onde quebra

Duas funções com nomes parecidos, semânticas diferentes, no mesmo arquivo público de um pacote Go compartilhado por duas equipes — na integração isso é, na melhor das hipóteses, um conflito de merge óbvio (nomes colidem ou quase colidem, alguém percebe no code review). Na pior hipótese, se apenas uma sobrevive à integração e o painel do Administrador (Pós-venda) acaba usando a função pensada para o pagamento (`EhTerminalParaPagamento`, que trata `PAGO` como terminal): `GET /admin/pedidos a cada 10s enquanto listar Pedido não terminal` (AD-18) para de reconsultar assim que o primeiro Pedido da lista chega a `PAGO`, mesmo que ele ainda vá evoluir para `SEPARANDO`/`ENVIADO`/`ENTREGUE`. O Administrador para de ver atualização automática de status logo depois do pagamento — precisa recarregar a página manualmente para ver o pedido avançar, o que é exatamente o tipo de "relógio que para de contar sozinho" que o Prevents do AD-18 tenta evitar (ainda que ali a frase seja sobre duração, o mecanismo do bug é irmão: estado que devia ser dinâmico congela por engano).

### Rule que fecha o buraco

> **AD-18, aperto.** Os estados terminais do Pedido são exatamente `{ENTREGUE, CANCELADO}` — a única leitura consistente com a tabela do AD-3 (nenhuma linha os lista como origem). Existe uma função só, `pedido.EstadoTerminal(s Status) bool`, em `pedido.go`, usada pelas três superfícies de polling do AD-18 sem exceção. Nenhum outro nome (`EhTerminalParaPagamento` ou equivalente) é declarado — quem precisa de um corte mais estreito do fluxo (por exemplo, "saiu da etapa de pagamento") testa `Status` diretamente contra os valores nomeados, nunca cria uma segunda noção de "terminal".
