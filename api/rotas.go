// Package api é só tradução: rota, middleware, DTO e os limites do NFR-14.
// Nenhuma regra de domínio mora aqui, e nenhum módulo de domínio conhece HTTP.
package api

import (
	"encoding/json"
	"mime"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Cashnip/amazon-waddle/internal/plataforma"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// servidor carrega as dependências que os handlers precisam. É struct, e não
// variável de pacote, para que o teste monte a sua sem tocar em estado global.
type servidor struct {
	cfg  plataforma.Config
	pool *pgxpool.Pool
	rdb  *redis.Client
}

// Rotas monta o ServeMux da biblioteca padrão. Dois handlers no mesmo padrão
// fazem o binário entrar em pânico no arranque — falha de subida, e não de
// comportamento, que é como o AD-16 quer a posse de rota.
func Rotas(cfg plataforma.Config, pool *pgxpool.Pool, rdb *redis.Client) http.Handler {
	s := &servidor{cfg: cfg, pool: pool, rdb: rdb}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/saude", saude)
	mux.HandleFunc("GET /api/v1/media/{arquivo}", midia)
	// A Sessão é o recurso criado, e por isso o plural do POST. O singular lê
	// a Sessão corrente — é o que o menu da conta da Épica 2 reaproveita.
	mux.HandleFunc("POST /api/v1/sessoes", s.criarSessao)
	mux.HandleFunc("GET /api/v1/sessao", s.lerSessao)
	// Encerrar é apagar o recurso: o singular já é a Sessão corrente, e o
	// DELETE apaga a chave no Redis — o cookie sozinho não invalida nada.
	mux.HandleFunc("DELETE /api/v1/sessao", s.encerrarSessao)
	// O Comprador é o recurso criado, e por isso o plural — o mesmo padrão de
	// `sessoes`. Cadastrar já abre a Sessão: o visitante sai autenticado.
	mux.HandleFunc("POST /api/v1/compradores", s.criarComprador)
	// A redefinição é o recurso: o POST a solicita (e sempre responde 202, para
	// não enumerar contas), e o PUT sobre o token a consome. O token está no
	// caminho, e não no corpo, porque é o próprio link que chega ao Comprador.
	mux.HandleFunc("POST /api/v1/redefinicoes-de-senha", s.criarRedefinicao)
	mux.HandleFunc("PUT /api/v1/redefinicoes-de-senha/{token}", s.redefinirSenha)
	// A listagem da loja é de `busca` (AD-16), e o detalhe é de `catalogo`.
	mux.HandleFunc("GET /api/v1/produtos", s.listarProdutos)
	mux.HandleFunc("GET /api/v1/produtos/{id}", s.detalheDoProduto)
	// As Categorias da loja, para o filtro: de `catalogo`, sem Sessão. É o
	// mesmo handler da listagem administrativa — a Categoria não tem nada que
	// o visitante não possa ver.
	mux.HandleFunc("GET /api/v1/categorias", s.listarCategorias)
	// Da Página de Produto direto ao Pedido, sem Carrinho (Épica 4). A leitura
	// é a tela do Pedido em processamento, consultada a cada 3 s.
	mux.HandleFunc("POST /api/v1/pedidos", s.criarPedido)
	mux.HandleFunc("GET /api/v1/pedidos/{id}", s.lerPedido)
	// "Meus pedidos" (2.6), no molde de "Escolher" Endereço: a lista do dono,
	// mais recente primeiro — o esboço que a Estória 6.1 substitui.
	mux.HandleFunc("GET /api/v1/pedidos", s.listarPedidos)
	// Os Endereços do Comprador (FR-5), no mux raiz: são da loja, e o dono é
	// quem a Sessão diz. "Escolher" é a listagem — não há Endereço padrão, e a
	// seleção do checkout é da 5.2.
	mux.HandleFunc("GET /api/v1/enderecos", s.listarEnderecos)
	mux.HandleFunc("POST /api/v1/enderecos", s.criarEndereco)
	mux.HandleFunc("PUT /api/v1/enderecos/{id}", s.atualizarEndereco)
	mux.HandleFunc("DELETE /api/v1/enderecos/{id}", s.removerEndereco)
	// O Carrinho do Comprador (FR-16): um por dono, resolvido pela Sessão, e por
	// isso sem identificador na rota. O que tem identificador é o Item.
	mux.HandleFunc("POST /api/v1/carrinho/itens", s.adicionarAoCarrinho)
	mux.HandleFunc("DELETE /api/v1/carrinho/itens/{id}", s.removerItemDoCarrinho)
	// O webhook não é autenticado por Sessão: quem chama é o Provedor, e o que
	// o autentica é o segredo compartilhado no cabeçalho. A chave de
	// idempotência não autentica ninguém — ela é derivada do identificador do
	// Pedido, que o próprio Comprador conhece.
	mux.HandleFunc("POST /api/v1/webhooks/pagamento", s.receberConfirmacao)

	// A área administrativa tem mux próprio, e a guarda mora no prefixo. Rota
	// nova da Épica 3 entra no `admin` e herda a autorização sem ninguém
	// lembrar dela — um `if` por handler seria esquecido no primeiro handler
	// escrito com pressa.
	admin := http.NewServeMux()
	admin.HandleFunc("GET /api/v1/admin/sessao", s.lerSessaoAdministrador)
	// A gestão de Vendedores (3.1): criar, editar, desativar e remover.
	admin.HandleFunc("GET /api/v1/admin/vendedores", s.listarVendedores)
	admin.HandleFunc("POST /api/v1/admin/vendedores", s.criarVendedor)
	admin.HandleFunc("PUT /api/v1/admin/vendedores/{id}", s.atualizarVendedor)
	admin.HandleFunc("DELETE /api/v1/admin/vendedores/{id}", s.removerVendedor)
	// A gestão de Categorias (3.2): criar, renomear e remover.
	admin.HandleFunc("GET /api/v1/admin/categorias", s.listarCategorias)
	admin.HandleFunc("POST /api/v1/admin/categorias", s.criarCategoria)
	admin.HandleFunc("PUT /api/v1/admin/categorias/{id}", s.renomearCategoria)
	admin.HandleFunc("DELETE /api/v1/admin/categorias/{id}", s.removerCategoria)
	// A gestão de Produtos (3.3): criar, editar, desativar e reativar — nunca
	// remover (FR-9). A listagem é administrativa e do `catalogo` (AD-16); a
	// da loja continua sendo de `busca`.
	admin.HandleFunc("GET /api/v1/admin/produtos", s.listarProdutosAdmin)
	admin.HandleFunc("POST /api/v1/admin/produtos", s.criarProduto)
	admin.HandleFunc("PUT /api/v1/admin/produtos/{id}", s.atualizarProduto)
	// O ajuste do Estoque total (3.4), em rota própria e com a guarda das
	// Reservas ativas.
	admin.HandleFunc("PUT /api/v1/admin/produtos/{id}/estoque", s.ajustarEstoque)
	// As imagens que um Produto pode usar: as de media/, embutidas.
	admin.HandleFunc("GET /api/v1/admin/midias", listarMidias)
	admin.HandleFunc("/", naoEncontrado)
	guardado := s.somenteAdministrador(admin)
	mux.Handle("/api/v1/admin/", guardado)
	// E o caminho exato, sem a barra: o ServeMux redirecionaria `/api/v1/admin`
	// para `/api/v1/admin/` com um 307 de corpo HTML, ANTES da guarda — um
	// chamador anônimo confirmaria a subárvore (contra a UX-DR9) e a API teria
	// um segundo contrato de erro (contra o AD-14). Atrás da MESMA guarda ele
	// cai no 404 do envelope, como qualquer outra rota administrativa.
	mux.Handle("/api/v1/admin", guardado)
	// O login do Administrador fica FORA da própria guarda, no mux raiz: padrão
	// mais específico ganha do prefixo (Go 1.22+). Sem isto ninguém entraria —
	// a guarda pediria a Sessão que só este handler sabe emitir.
	mux.HandleFunc("POST /api/v1/admin/sessoes", s.criarSessaoAdministrador)

	// Sem isto o ServeMux responderia "404 page not found" em texto puro, e a
	// API teria dois contratos de erro conforme a rota exista ou não (AD-14).
	mux.HandleFunc("/", naoEncontrado)
	return plataforma.Correlacao(mux)
}

func naoEncontrado(w http.ResponseWriter, r *http.Request) {
	erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
}

// decodificarCorpo é o único lugar que lê corpo JSON de requisição. Junta as
// duas guardas que toda rota com corpo precisa: o teto do NFR-14 e a exigência
// do `Content-Type: application/json`.
//
// O cabeçalho é o que fecha a fixação de Sessão por formulário cross-site: um
// <form> só consegue emitir text/plain, x-www-form-urlencoded ou
// multipart/form-data — application/json exigiria fetch, e aí o preflight do
// CORS entra na frente. A recusa mora aqui, junto do MaxBytesReader, e não num
// middleware: GET e DELETE não têm corpo e não podem passar a exigir cabeçalho.
//
// Devolve ErrEntradaInvalida em todos os casos; quem chama decide o que
// escrever — a solicitação de redefinição, por exemplo, responde o mesmo 202
// de sempre, porque distinguir os caminhos ali enumeraria contas.
func decodificarCorpo(w http.ResponseWriter, r *http.Request, destino any) error {
	return decodificarCorpoAte(w, r, destino, corpoMaximo)
}

// decodificarCorpoAte é o decodificarCorpo com outro teto de bytes — hoje só o
// Produto, cuja descrição de milhares de caracteres não cabe nos 4 KiB.
func decodificarCorpoAte(w http.ResponseWriter, r *http.Request, destino any, maximo int64) error {
	// ParseMediaType e não comparação crua: `application/json; charset=utf-8` é
	// o mesmo tipo, e um chamador legítimo que mande o parâmetro não pode ser
	// recusado.
	if tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || tipo != "application/json" {
		return erro.ErrEntradaInvalida
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maximo)).Decode(destino); err != nil {
		return erro.ErrEntradaInvalida
	}
	return nil
}

// escreverJSON é o único lugar que serializa resposta de sucesso. O status é
// parâmetro porque o 201 do Pedido precisa dele, e escrever o cabeçalho fora
// daqui perderia o Content-Type — WriteHeader congela o que já foi posto. O
// erro de codificação não vira envelope: o cabeçalho já foi, e o cliente já
// está lendo.
func escreverJSON(w http.ResponseWriter, status int, valor any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(valor)
}

// listagem é o envelope do AD-18, o mesmo para a loja e para o Administrador.
type listagem[T any] struct {
	Itens     []T   `json:"itens"`
	Pagina    int   `json:"pagina"`
	PorPagina int   `json:"por_pagina"`
	Total     int64 `json:"total"`
}

// paginacaoDe lê `pagina` e `por_pagina` da URL. Ausentes, valem 1 e o padrão
// da Config; `por_pagina` acima do teto é rebaixado ao teto, e não recusado.
// Não inteiro ou menor que 1 escreve o erro em linha e devolve ok falso.
func paginacaoDe(w http.ResponseWriter, r *http.Request, cfg plataforma.Config) (pagina, porPagina int, ok bool) {
	ler := func(campo, mensagem string, padrao int) (int, bool) {
		texto := r.URL.Query().Get(campo)
		if texto == "" {
			return padrao, true
		}
		n, err := strconv.Atoi(texto)
		if err != nil || n < 1 {
			erro.EscreverCampo(r.Context(), w, campo, mensagem)
			return 0, false
		}
		return n, true
	}
	if pagina, ok = ler("pagina", "A página é um número inteiro a partir de 1.", 1); !ok {
		return 0, 0, false
	}
	if porPagina, ok = ler("por_pagina", "Os itens por página são um número inteiro a partir de 1.", cfg.PaginaTamanho); !ok {
		return 0, 0, false
	}
	return pagina, min(porPagina, cfg.PaginaTamanhoMax), true
}
