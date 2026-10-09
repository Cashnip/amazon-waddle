---
title: '8.2 — Os três roteiros ensaiados de novo sobre o catálogo novo'
type: 'chore'
created: '2026-10-08'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
baseline_commit: '04d800f2ae3ce194be66dc69d9bf051d4feec920'
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-8-context.md'
  - '{project-root}/_bmad-output/implementation-artifacts/spec-7-4-os-tres-roteiros-ensaiados-a-partir-de-clone-limpo.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A 8.1 trocou o catálogo, e com ele os Produtos que provocam cada desfecho. Isso deixou desatualizados o ensaio de 2026-09-28, o `roteiro-da-apresentacao.md` e o `HANDOFF.md`. Nada pode ser demonstrado ao vivo pela primeira vez (SM-1).

**Approach:** Repetir o ensaio da 7.4 sobre o catálogo novo: os três roteiros do §12, de ponta a ponta, **no navegador**, a partir de clone limpo de `main`. A execução fica registrada numa seção datada no fim de `ensaio-dos-roteiros.md`. Depois, o roteiro da apresentação e o HANDOFF são atualizados. A estória só fecha com o ensaio feito por uma pessoa (checkpoint humano).

**Decidido em 2026-10-08 (NFR-15):** o agente ensaia online e, no Roteiro A, prova que nada sai de `localhost` pelas requisições de rede do navegador (nenhuma requisição a host externo, fotos inclusive). O Roteiro A **com a rede desligada** é feito pela pessoa no checkpoint e vai para a linha dela no Registro; a seção do agente diz isso, sem declarar o NFR-15 cumprido.

## Boundaries & Constraints

**Always:**
- **Ambiente:** clone limpo de `main` (da origin) em `C:\dev\ensaio82` e `docker compose up --build`, conferido em `docker compose images`.
- **Sem intervenção no banco:** nada de `psql` e nada reconfigurado entre roteiros. Intervenção no banco reprova o roteiro.
- **Pilha de trabalho:** a do repositório ocupa as portas 3000 e 8080. Ela é parada com `docker compose stop` (sem `-v`) antes do ensaio e religada com `docker compose start` depois.
- **Evidência:** é o que a tela mostrou, com o status e o horário. Passo não executado fica marcado como **não ensaiado**, nunca como aprovado.
- **Perfis de navegador:** um por papel, por causa do defeito 1 da 7.4.
- **Desmontagem do clone ao fim:** `down -v`, `docker rmi` das duas imagens do ensaio e o diretório apagado.
- **Registro do ensaio:** é uma seção nova `## Ensaio de 2026-10-08 — catálogo da 8.1` no fim de `ensaio-dos-roteiros.md`, com contagem, ambiente, as tabelas dos três roteiros no formato da 7.4 e um Registro próprio. Esse Registro tem a linha do agente ("ensaio técnico, não conta como uma das duas pessoas") e uma linha **pessoa** em branco.
- **Arquivo preservado:** tudo o que já existe acima da seção nova fica intacto, as duas linhas "em aberto" do Registro de 2026-09-28 inclusive.

**Ask First:** Qualquer defeito cuja correção mude código de produção. Nesse caso, HALT com a lista antes de tocar em arquivo fora dos entregáveis.

**Never:**
- Corrigir defeito de comportamento aqui. Defeito vira entrada em `deferred-work.md`.
- Funcionalidade nova.
- Declarar cumprida a metade humana: o agente não é a pessoa da condição de aceite.
- Reescrever o §12 do PRD ou as seções antigas do ensaio.

## I/O & Edge-Case Matrix

| Cenário | Dado da demonstração | Esperado na tela |
|---|---|---|
| Roteiro A (passos 1–9) | busca `mao`, sem acento, que acha **Mixer de Mão** e outros nomes com "Mão"; faixa de R$ 10 a R$ 500; ordenar por menor preço; Mixer de Mão R$ 192,00, 1 un., CEP `01310-100` → **total R$ 207,00** (Sul `80010-000` → R$ 212,00) | o mesmo dos 9 passos da 7.4, com as fotos carregando |
| Roteiro B passos 1–4 | Carregador de Celular com Cabo R$ 110,90 → **R$ 125,90** | `PAGAMENTO_RECUSADO` com o motivo; o Estoque volta em outra aba; "Tentar pagar de novo" chega a `PAGO` |
| Roteiro B passo 5 | Bola de Futebol R$ 99,95 → **R$ 114,95** | sai sozinho em 60 s com "Tempo de pagamento expirado." e o Estoque volta |
| Roteiro C passos 1–5 | os da 7.4, com Pedidos `PAGO` novos do Mixer; o passo 5 só entra com o teste da NFR-7 verde no clone | os da 7.4 |
| Erro visível | notificação com identificador de correlação | o roteiro **reprova**; o defeito é registrado, não corrigido |

</frozen-after-approval>

## Code Map

- `_bmad-output/implementation-artifacts/ensaio-dos-roteiros.md` -- é o formato a imitar: contagem, ambiente, tabelas `# · FR · Dado · O que tem de aparecer · Evidência · Veredito`, defeitos e Registro. A seção nova é acrescentada no fim.
- `_bmad-output/implementation-artifacts/roteiro-da-apresentacao.md` -- é o roteiro de 15 minutos.
  - Linhas 102 e 107–135 citam Maré, Aurora e Andarilho, com preços e totais.
  - A linha 122 tem o `mare`.
  - As linhas 161–162 são a preparação do Roteiro B.
  - Só os Produtos, preços, totais e termo mudam; a estrutura não muda.
- `HANDOFF.md` -- o cabeçalho "Estado" passa a apontar a Épica 8: o catálogo da 8.1 e o ensaio da 8.2.
- `README.md`, seção 2 -- a frase de 8.1 diz que os tempos foram medidos em 2026-09-25 e que "a 8.2 mede de novo". Ela troca a promessa pela referência aos tempos deste ensaio.
- **Papéis do catálogo**, em `media/gerar.go` e no README:
  - Mixer `7446cd4e-bfa7-5bbe-8929-c987cc39a1d5`;
  - Carregador `71ea1e58-91d9-5022-ba93-6a2518f42eed`;
  - Bola `0faf8e31-3b0c-5a47-8c00-34d462d42aa6`.
- **Faixas do Simulado:** `,00`–`,89` aprova, `,90`–`,94` recusa, `,95`–`,99` expira. O Frete é de R$ 15 no Sudeste e R$ 20 no Sul, e é grátis a partir de R$ 299,00.
- **Teste da NFR-7:** `go test ./api -run TestSessaoEProduto -v`, que agora roda nativo nesta máquina.

## Tasks & Acceptance

**Execution:**
- [x] **Ambiente** -- Parar a pilha de trabalho; clonar em `C:\dev\ensaio82`; `up --build`; rodar `go test ./...` e o teste da NFR-7 no clone.
- [x] **Navegador** -- Executar os Roteiros A, B e C na sequência, sem reinício, anotando a evidência com o horário.
- [x] `ensaio-dos-roteiros.md` -- Escrever a seção datada no fim: contagem, ambiente, três tabelas, o que não cobriu, defeitos e Registro com a linha do agente e a linha da pessoa em branco.
- [x] `deferred-work.md` -- Registrar cada defeito achado, se houver.
- [x] `roteiro-da-apresentacao.md` -- Pôr os Produtos, preços, totais e o termo `mao` novos, mais as armadilhas de quantidade (duas Bolas dão R$ 214,90 e recusam) e os horários da tabela, ajustados ao que o ensaio viu.
- [x] `README.md` -- Trocar "a 8.2 mede de novo" pelos tempos deste ensaio, com o link da seção.
- [x] `HANDOFF.md` -- Fazer o cabeçalho apontar o catálogo novo e o ensaio de 2026-10-08, com a metade humana pendente.
- [x] **Desmontagem** -- Limpar o clone, as imagens e o volume; religar a pilha de trabalho.

**Acceptance Criteria:**
- Given o ensaio, when a seção nova é lida, then cada passo dos três roteiros tem veredito e evidência com horário, e a contagem fecha.
- Given `git diff` em `ensaio-dos-roteiros.md`, then só há linhas acrescentadas depois da última linha antiga.
- Given o fim da estória, then nenhum contêiner, imagem ou volume `ensaio82*` sobra, e a pilha de trabalho está de pé.

## Implementation Notes

## Spec Change Log

## Review Triage Log

Passe 1 (Blind Hunter, Edge Case Hunter, Verification Gap — esta sem achados, o diff é só documentação):

| # | Lente | Achado | Veredito | Evidência / rota |
|---|---|---|---|---|
| 1 | BH, ECH | O teste da NFR-7 "terminou às 22:48:51, antes do primeiro passo do navegador", mas o Roteiro A começou 22:48:09 | low | Linhas 355–357 contra 415 e 437; a porta que importa (antes do passo 5 do C) vale → patch (texto) |
| 2 | BH, ECH | Reabertura dos perfis "entre o Roteiro B e o C", mas o C passo 1 correu 23:00–23:01 | low | Linha 390 contra a tabela do C → patch (texto) |
| 3 | BH, ECH | A tabela de rede rotula "Roteiros B e C" uma janela que só cobre 23:04–23:08; B e C1 não foram registrados | medium | Linha 416; o texto deixa crer que o tráfego do B foi conferido → patch (rótulo + lacuna declarada) |
| 4 | BH | O Roteiro B correu na ordem 1, 2, 4, 3 sem dizer | low | Passo 4 às 22:56:32, segunda tentativa do 3 às 22:57:14 → patch (uma frase) |
| 5 | BH | A busca `mao` não explica o Porta-Temperos, e o roteiro da apresentação cita 4 dos 6 resultados sem o Limão | low | A busca lê nome e descrição ("sempre à mão"); correção direta → patch |
| 6 | BH, ECH | O caminho do roteiro da apresentação (Confirmar com Frete Sul R$ 212,00; cancelar o Pedido do Carregador `PAGO`) não foi o ensaiado | low | O ensaio confirmou em SP e cancelou o Mixer no C3; falta dizer em "O que este ensaio não cobriu" → patch |
| 7 | BH | O HANDOFF mantém a Épica 7 "em `review`" e o "Próximo passo" na Épica 7 | medium | `HANDOFF.md:10` e a seção da linha 71; a Épica 7 fechou em `acf3de7`; quem abrir o HANDOFF não acha o checkpoint da 8.2 → patch |
| 8 | BH | "com **24 s**" lido como tamanho ou duração | low | Correção direta: "criada há 24 s" → patch |
| 9 | BH, ECH | A tarefa promete "os horários da tabela ajustados", e a coluna Minuto não mudou | false | O que o ensaio mudou foi o "~7 s" → "~6 s", que mudou; os minutos do roteiro não dependem do catálogo |
| 10 | ECH | Dois Carregadores aprovam (R$ 236,80) | false | `roteiro-da-apresentacao.md` já diz "Leve sempre **uma** unidade nos caminhos 2 e 3" |
| 11 | BH | O C passo 5 para no Pedido em `AGUARDANDO_PAGAMENTO`, sem ler o Estoque esgotado | false | A prova do "um Pedido só" é o segundo Comprador recebendo "Estoque insuficiente"; é o mesmo critério da 7.4 |
| 12 | BH, ECH | Spec `in-review` e sprint `in-progress`; `last_updated` anterior ao ensaio | false | A sincronização é do passo 5 |
| 13 | BH | O comando de Verification não confere imagens e redes | low | O conserto edita a spec → rejeitado; o ensaio registra os quatro comandos e conferi `docker images` vazio |
| 14 | BH | Implementation Notes vazias apesar dos desvios (CDP, reabertura) | low | O conserto edita a spec, e os desvios estão no próprio ensaio → rejeitado |
| 15 | BH | A entrada do NFR-15 em `deferred-work.md` não cita a metade online | low | A entrada continua exata (o offline não foi refeito), e a seção nova a cita; o checkpoint humano a fecha → rejeitado |

## Verification

**Commands:**
- `git diff --stat` -- só os arquivos de documentação e rastreamento listados nas Tasks; nenhum arquivo de produção.
- `docker ps -a --format '{{.Names}}' | grep ensaio82 ; docker volume ls | grep ensaio82` -- os dois saem vazios.

**Manual checks:**
- O ensaio da pessoa, no checkpoint, com a linha dela preenchida no Registro da seção nova.
