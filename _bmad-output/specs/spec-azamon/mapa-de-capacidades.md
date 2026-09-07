---
title: Azamon — Mapa de Capacidades
status: final
updated: 2026-09-06
---

# Mapa de Capacidades

Companion de [`SPEC.md`](SPEC.md). Cada `CAP-N` do kernel amarrado às cinco coisas que os documentos vivos dizem sobre ele em lugares diferentes: quais Requisitos Funcionais a compõem, onde ela mora no código, quais invariantes de arquitetura a governam, quais superfícies de interface a realizam, e onde ela é demonstrada. É a junção que não existe em nenhum documento de origem — quem vai escrever épicas fatia por aqui em vez de reler seis arquivos.

Os IDs `CAP-N` são estáveis e nunca reutilizados. `FR`, `NFR`, `SM` e `UJ` vêm do PRD; `AD` vem da espinha de arquitetura; os roteiros vêm do §12 do PRD.

## A tabela

| CAP | Requisitos | Módulo | Governada por | Superfícies | Demonstrada em | Passo do addendum §8 |
|---|---|---|---|---|---|---|
| **CAP-1** Conta e Identidade | FR-1..FR-5 | `internal/identidade` + Redis | AD-11, AD-13, AD-20 | Login · Cadastro · Recuperação de senha · Meus endereços · Perfil | A2 | 1 |
| **CAP-2** Catálogo e Estoque | FR-6..FR-11 | `internal/catalogo` | AD-2, AD-5, AD-9, AD-12, AD-19 | Vitrine · Página de Produto · Vendedores · Categorias · Produtos | A4 · B3 · C1 | 2 |
| **CAP-3** Busca e Navegação | FR-12..FR-15 | `internal/busca` (VIEW sobre `catalogo`) | AD-16, AD-2, AD-19 | Resultados de busca · Faixa de Categorias | A3 | 6 — paralelizável a partir do passo 2 |
| **CAP-4** Carrinho | FR-16..FR-19 | `internal/carrinho` | AD-5, AD-9, AD-11, AD-17, AD-19 | Carrinho | A5 | 4 |
| **CAP-5** Checkout e pagamento | FR-20..FR-27, FR-34 | `internal/pedido` + `internal/pagamento` | AD-3..AD-9, AD-17, AD-18 | Checkout (Endereço → Revisão) · Pedido em processamento | A6 · A7 · B1..B5 · C5 (NFR-7) | 5 |
| **CAP-6** Pedido e pós-venda | FR-28..FR-33 | `internal/pedido` + varredura | AD-3, AD-4, AD-6, AD-15, AD-18 | Meus pedidos · Detalhe do Pedido · Pedidos (Administrador) | A8 · A9 · C2..C4 | **3** para a máquina de estados, **7** para o resto |
| **CAP-7** Ambiente em um comando | NFR-1, NFR-15 | `cmd/azamon` · `db/` · `media/` · `docker-compose.yml` | AD-12, AD-13 | — | A1 | 0 |
| **CAP-8** Entregáveis de documentação | §6.1, SM-7 | `README.md` · `DIAGRAMA-MODULOS.md` · `addendum.md` | AD-1 | — | — | contínuo |

**CAP-6 é a única capacidade partida em dois passos, e a divisão é deliberada.** A máquina de estados (FR-28) é pré-requisito de quatro funcionalidades e de dois dos três roteiros: ela é construída **com testes, antes das telas que a consomem**. O que ela sustenta — histórico, detalhe, cancelamento, painel e simulação — vem no fim, e sai quase de graça.

## Requisitos transversais

Sete NFRs não pertencem a nenhuma capacidade e cobram de todas. Uma épica que os trate como trabalho de alguém no fim do semestre é uma épica que os perde.

| Requisito | O que cobra de toda capacidade | Onde vive a regra |
|---|---|---|
| NFR-2 | Um módulo só é alcançado pela sua interface pública; o grafo de importação é verificado por teste | AD-1, `internal/fronteira_test.go` |
| NFR-8 | Todo FR do fluxo busca → Carrinho → checkout → Pedido tem teste que falha se a consequência declarada quebrar | Convenções de Consistência, SM-2 |
| NFR-9 | Correlação em toda linha de log, e toda transição de Status registrada | AD-15 |
| NFR-10 · NFR-11 | 360–1440 px sem rolagem horizontal; WCAG 2.2 AA completo | `EXPERIENCE.md`, `DESIGN.md` — de propósito sem AD |
| NFR-13 | Centavos inteiros ponta a ponta; o total exibido fecha com as parcelas exibidas | AD-9 |
| NFR-14 | Limite declarado e verificado no servidor para todo campo e parâmetro | AD-13, aplicado em `api/` |
| NFR-16 | Todo limiar em configuração, nunca constante espalhada | AD-13, §7.1 do PRD |

## Ordem de corte, mapeada em capacidades

O §6.3 do PRD corta **nesta ordem** se o time for o extremo menor ou se a SM-6 não for atingida:

1. FR-3 Recuperação de senha → **CAP-1**
2. FR-10 Gestão de Categorias → **CAP-2**
3. FR-8 Gestão de Vendedores → **CAP-2**
4. FR-33 Simulação de entrega → **CAP-6**
5. FR-14 Ordenação → **CAP-3**

**Nunca cortar, por mais invisíveis que pareçam:** FR-19 (**CAP-4**), FR-24 e FR-34 (**CAP-5**) e o teste de concorrência do NFR-7. São o que separa o sistema da maquete, e nenhum deles aparece numa tela.

Nenhuma capacidade some inteira na ordem de corte: o corte é de requisito, e as oito continuam existindo.
