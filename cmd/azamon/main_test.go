package main

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/pagamento"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// expiracaoDeTesteDoBinario é o AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO com que o
// binário sobe nestes testes: 6 s contra um tique de 1 s. Os 15 min de produção
// (ou os 60 s da demonstração) fariam o Pedido recente sobreviver ao teste
// inteiro sem provar nada, e um valor perto do tique não distinguiria "o prazo
// é este" de "o prazo é qualquer coisa curta".
const expiracaoDeTesteDoBinario = 6 * time.Second

// A ordem do arranque é normativa: sem banco, o processo não chega a escutar.
func TestExecutarNaoServeSemBanco(t *testing.T) {
	const endereco = "127.0.0.1:8099"
	t.Setenv("AZAMON_HTTP_ADDR", endereco)
	t.Setenv("AZAMON_POSTGRES_DSN", "postgres://azamon@127.0.0.1:1/azamon")
	t.Setenv("AZAMON_REDIS_URL", "redis://127.0.0.1:1/0")
	t.Setenv("AZAMON_WEBHOOK_SEGREDO", "segredo-de-teste")

	ctx, cancelar := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelar()

	if err := executar(ctx, io.Discard); err == nil {
		t.Fatal("executar devia falhar sem banco")
	}
	c, err := net.DialTimeout("tcp", endereco, 100*time.Millisecond)
	if err == nil {
		c.Close()
		t.Fatalf("%s ficou escutando; a migração tem de barrar o servidor", endereco)
	}
}

// Sem este teste, apagar a chamada a Semear de executar() deixa `go test ./...`
// inteiro verde: main_test falha dentro de Migrar e nunca chega lá, e
// db/schema_test chama Semear direto. A demonstração é que subiria com a loja
// vazia. Aqui o arranque roda inteiro, contra um Postgres de verdade.
func TestExecutarDeixaOCatalogoSemeado(t *testing.T) {
	const endereco = "127.0.0.1:8098"
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx, "postgres:18.6",
		tcpostgres.WithDatabase("azamon"),
		tcpostgres.WithUsername("azamon"),
		tcpostgres.WithPassword("azamon"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(2*time.Minute)),
	)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(ctr) })
	if err != nil {
		t.Fatalf("subir postgres: %v", err)
	}
	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("DSN do contêiner: %v", err)
	}

	t.Setenv("AZAMON_HTTP_ADDR", endereco)
	t.Setenv("AZAMON_POSTGRES_DSN", dsn)
	t.Setenv("AZAMON_REDIS_URL", "redis://127.0.0.1:1/0")
	t.Setenv("AZAMON_WEBHOOK_SEGREDO", "segredo-de-teste")
	// O webhook do Provedor Simulado volta para o próprio binário, que é como
	// ele roda em compose. Com isso o tique, o transporte HTTP de verdade, a
	// rota e a varredura entram todos no caminho do teste.
	t.Setenv("AZAMON_WEBHOOK_BASE_URL", "http://"+endereco)
	t.Setenv("AZAMON_CONFIRMACAO_ATRASO", "100ms")
	// Em demonstração a etapa é de 30 s; aqui o que importa é que ela exista e
	// seja lida da configuração, então encolhe para o teste caber num tique.
	t.Setenv("AZAMON_ENTREGA_INTERVALO", "200ms")
	// O prazo da FR-34, encolhido pelo mesmo motivo. Tem de ser bem maior que
	// o tique (1 s) e bem maior que o intervalo da entrega, senão um prazo
	// trocado por engano passaria despercebido; e maior que
	// AZAMON_CONFIRMACAO_ATRASO, que a Config exige.
	t.Setenv("AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO", expiracaoDeTesteDoBinario.String())

	servindo, parar := context.WithCancel(ctx)
	fim := make(chan error, 1)
	go func() { fim <- executar(servindo, io.Discard) }()
	// Desligar é explícito e não só adiado: o subteste do interruptor precisa
	// que este binário já tenha parado antes de o próximo subir, senão a
	// varredura dele avançaria o Pedido que o outro espera parado.
	parado := false
	pararEsperando := func() {
		if parado {
			return
		}
		parado = true
		parar()
		<-fim
	}
	defer pararEsperando()

	// Escutar é o sinal de que migração e semente passaram: a ordem do
	// arranque é config → migrações → semente → servir.
	prazo := time.Now().Add(time.Minute)
	for {
		if c, err := net.DialTimeout("tcp", endereco, time.Second); err == nil {
			c.Close()
			break
		}
		if time.Now().After(prazo) {
			t.Fatal("executar não chegou a escutar")
		}
		time.Sleep(50 * time.Millisecond)
	}

	conexao, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("conectar: %v", err)
	}
	defer conexao.Close(ctx)
	var n int
	if err := conexao.QueryRow(ctx, `SELECT count(*) FROM catalogo.produto`).Scan(&n); err != nil {
		t.Fatalf("contar Produtos: %v", err)
	}
	if n == 0 {
		t.Error("o arranque terminou com o catálogo vazio; Semear não rodou")
	}

	pedidoID, produtoID, estoqueAntes := varreduraLevaOPedidoAPago(t, ctx, conexao)
	simulacaoLevaOPedidoAEntregue(t, ctx, conexao, pedidoID, produtoID, estoqueAntes)
	oTiqueExpiraNaOrdemDoAD6(t, ctx, conexao)

	pararEsperando()
	deixados := interruptorDesligaASimulacao(t, ctx, conexao)
	reinicioRetomaAEntrega(t, ctx, conexao, deixados)
}

// deixado é um Pedido vencido que o binário com a simulação desligada deixa
// para trás, com a Reserva ATIVA de um Produto só dele e o Estoque total de
// antes — o que o arranque seguinte tem de retomar.
type deixado struct {
	pedido, produto string
	estoqueAntes    int32
	// avancos é quantas linhas SIMULACAO o caminho até ENTREGUE grava.
	avancos int
}

// deixarVencido monta o Pedido por INSERT, com o histórico de uma hora atrás
// e a Reserva ATIVA de 1 unidade do Produto de posição `produto` na ordem de
// id. Posições acima de 0: a 0 é a dos outros subtestes deste arquivo, que
// contam o Estoque dela. O histórico entra por último, no mesmo commit — antes
// dele o Pedido não é candidato de tique nenhum.
func deixarVencido(t *testing.T, ctx context.Context, conexao *pgx.Conn, numero string, produto int, passos ...[3]string) deixado {
	t.Helper()
	tx, err := conexao.Begin(ctx)
	if err != nil {
		t.Fatalf("abrir a transação de %s: %v", numero, err)
	}
	defer tx.Rollback(ctx)
	d := deixado{avancos: 3}
	if passos[len(passos)-1][1] == "SEPARANDO" {
		d.avancos = 2
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO pedido.pedido (numero, comprador_id, status, subtotal_centavos, frete_centavos, total_centavos)
		VALUES ($1, uuidv7(), $2, 32900, 0, 32900)
		RETURNING id::text`, numero, passos[len(passos)-1][1]).Scan(&d.pedido); err != nil {
		t.Fatalf("criar o Pedido %s: %v", numero, err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT id::text, estoque_total FROM catalogo.produto ORDER BY id OFFSET $1 LIMIT 1`, produto).
		Scan(&d.produto, &d.estoqueAntes); err != nil {
		t.Fatalf("escolher o Produto de %s: %v", numero, err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO catalogo.reserva_estoque (produto_id, pedido_id, quantidade, estado)
		VALUES ($1::uuid, $2::uuid, 1, 'ATIVA')`, d.produto, d.pedido); err != nil {
		t.Fatalf("criar a Reserva de %s: %v", numero, err)
	}
	for i, p := range passos {
		if _, err := tx.Exec(ctx, `
			INSERT INTO pedido.transicao_status (pedido_id, status_anterior, status_novo, ator, ocorrido_em)
			VALUES ($1::uuid, $2, $3, $4, now() - make_interval(mins => $5))`,
			d.pedido, p[0], p[1], p[2], 60+len(passos)-1-i); err != nil {
			t.Fatalf("registrar %s → %s de %s: %v", p[0], p[1], numero, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit de %s: %v", numero, err)
	}
	return d
}

// retratoDoDeixado é o Status, as linhas SIMULACAO do histórico, o estado da
// Reserva e o Estoque total do Produto, numa leitura só.
func retratoDoDeixado(t *testing.T, ctx context.Context, conexao *pgx.Conn, d deixado) (status string, avancos int, reserva string, estoque int32) {
	t.Helper()
	if err := conexao.QueryRow(ctx, `
		SELECT p.status,
		       (SELECT count(*) FROM pedido.transicao_status h WHERE h.pedido_id = p.id AND h.ator = 'SIMULACAO'),
		       r.estado, pr.estoque_total
		FROM pedido.pedido p
		JOIN catalogo.reserva_estoque r ON r.pedido_id = p.id
		JOIN catalogo.produto pr ON pr.id = r.produto_id
		WHERE p.id = $1::uuid`, d.pedido).Scan(&status, &avancos, &reserva, &estoque); err != nil {
		t.Fatalf("ler o Pedido %s: %v", d.pedido, err)
	}
	return status, avancos, reserva, estoque
}

// reinicioRetomaAEntrega é o "reiniciar retoma de onde parou" do AD-6 medido
// no binário inteiro: o arranque novo encontra os Pedidos que o anterior
// deixou e os leva até ENTREGUE — o PAGO pelas três etapas, o SEPARANDO pelas
// duas que faltam, cada um consolidando a sua Reserva uma vez. Sem isto, um
// arranque que perdesse ou refizesse etapa deixaria tudo verde: os outros
// testes sobem um binário só e começam do zero. O que ele NÃO prova é que o
// instante do avanço sai do histórico — a 200 ms, um relógio em memória zerado
// no arranque também chegaria a tempo; essa prova é a de api/, com o histórico
// de uma hora vencendo o intervalo de meia na primeira chamada.
func reinicioRetomaAEntrega(t *testing.T, ctx context.Context, conexao *pgx.Conn, deixados []deixado) {
	t.Helper()
	// O binário desligado já parou: o que ele deixou está como foi montado,
	// e o que acontecer daqui em diante é do arranque novo.
	for _, d := range deixados {
		status, avancos, reserva, estoque := retratoDoDeixado(t, ctx, conexao, d)
		if avancos != 0 || reserva != "ATIVA" || estoque != d.estoqueAntes {
			t.Fatalf("o binário desligado mexeu no Pedido %s: %s, %d avanços, Reserva %s, estoque_total %d",
				d.pedido, status, avancos, reserva, estoque)
		}
	}

	const endereco = "127.0.0.1:8096"
	t.Setenv("AZAMON_HTTP_ADDR", endereco)
	t.Setenv("AZAMON_WEBHOOK_BASE_URL", "http://"+endereco)
	t.Setenv("AZAMON_ENTREGA_SIMULACAO_ATIVA", "true")
	t.Setenv("AZAMON_ENTREGA_INTERVALO", "200ms")

	servindo, parar := context.WithCancel(ctx)
	fim := make(chan error, 1)
	go func() { fim <- executar(servindo, io.Discard) }()
	defer func() { parar(); <-fim }()

	prazo := time.Now().Add(time.Minute)
	for {
		if c, err := net.DialTimeout("tcp", endereco, time.Second); err == nil {
			c.Close()
			break
		}
		if time.Now().After(prazo) {
			t.Fatal("o arranque da retomada não chegou a escutar")
		}
		time.Sleep(50 * time.Millisecond)
	}

	for _, d := range deixados {
		prazo := time.Now().Add(30 * time.Second)
		for {
			status, avancos, reserva, estoque := retratoDoDeixado(t, ctx, conexao, d)
			if status == "ENTREGUE" {
				if avancos != d.avancos {
					t.Errorf("Pedido %s: %d linhas SIMULACAO, quero %d", d.pedido, avancos, d.avancos)
				}
				if reserva != "CONSOLIDADA" || estoque != d.estoqueAntes-1 {
					t.Errorf("Pedido %s: Reserva %s e estoque_total %d; quero CONSOLIDADA e %d — uma consolidação, e só uma",
						d.pedido, reserva, estoque, d.estoqueAntes-1)
				}
				break
			}
			if time.Now().After(prazo) {
				t.Fatalf("o Pedido %s ficou em %s depois do arranque; a simulação não retomou de onde parou", d.pedido, status)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// interruptorDesligaASimulacao prova a consequência da FR-33 que nenhuma outra
// verificação alcança: a simulação pode ser desligada por configuração, para o
// Administrador conduzir a apresentação manualmente. Sem isto, apagar o
// `if cfg.EntregaSimulacaoAtiva` do tique deixa tudo verde.
//
// E prova o outro lado, que é da FR-34 🔒: o interruptor NÃO cobre a expiração.
// Com ele desligado, a Tentativa de Pagamento vencida continua expirando e a
// Reserva de Estoque continua voltando para a prateleira — mover
// `pedido.Expirar` para dentro daquele `if` faria a FR-34 depender de
// configuração em silêncio, e é esta metade que mata a mutação.
//
// A configuração é lida uma vez, no arranque, então o interruptor só é
// observável subindo o binário de novo — no mesmo Postgres, que é a parte cara.
//
// Devolve os dois Pedidos vencidos que este binário deixa ao parar — o PAGO
// do interruptor e um SEPARANDO —, que são o ponto de partida da retomada.
func interruptorDesligaASimulacao(t *testing.T, ctx context.Context, conexao *pgx.Conn) []deixado {
	t.Helper()
	const endereco = "127.0.0.1:8097"
	t.Setenv("AZAMON_HTTP_ADDR", endereco)
	t.Setenv("AZAMON_WEBHOOK_BASE_URL", "http://"+endereco)
	t.Setenv("AZAMON_VARREDURA_INTERVALO", "50ms")
	t.Setenv("AZAMON_ENTREGA_SIMULACAO_ATIVA", "false")

	// Um Pedido em PAGO com o histórico já vencido: com a simulação ligada,
	// este é exatamente o Pedido que o primeiro tique moveria.
	pago := deixarVencido(t, ctx, conexao, "AZ-INTERRUPTOR-0001", 1,
		[3]string{"", "AGUARDANDO_PAGAMENTO", "COMPRADOR"},
		[3]string{"AGUARDANDO_PAGAMENTO", "PAGO", "PROVEDOR"})
	pedidoID := pago.pedido

	servindo, parar := context.WithCancel(ctx)
	fim := make(chan error, 1)
	go func() { fim <- executar(servindo, io.Discard) }()
	defer func() { parar(); <-fim }()

	prazo := time.Now().Add(time.Minute)
	for {
		if c, err := net.DialTimeout("tcp", endereco, time.Second); err == nil {
			c.Close()
			break
		}
		if time.Now().After(prazo) {
			t.Fatal("o segundo arranque não chegou a escutar")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Vários tiques de 50 ms: tempo de sobra para a simulação agir, se
	// estivesse ligada.
	time.Sleep(2 * time.Second)
	var status string
	if err := conexao.QueryRow(ctx,
		`SELECT status FROM pedido.pedido WHERE id = $1::uuid`, pedidoID).Scan(&status); err != nil {
		t.Fatalf("ler o Status: %v", err)
	}
	if status != "PAGO" {
		t.Errorf("status = %s com a simulação desligada; quero PAGO parado", status)
	}

	// O segundo que este binário deixa: SEPARANDO vencido, com a Reserva por
	// consolidar. Nasce antes da expiração abaixo, que ainda roda vários
	// tiques com a simulação desligada — e a retomada confere que nenhum
	// deles o moveu.
	separando := deixarVencido(t, ctx, conexao, "AZ-INTERRUPTOR-0003", 2,
		[3]string{"", "AGUARDANDO_PAGAMENTO", "COMPRADOR"},
		[3]string{"AGUARDANDO_PAGAMENTO", "PAGO", "PROVEDOR"},
		[3]string{"PAGO", "SEPARANDO", "ADMINISTRADOR"})

	expiracaoIgnoraOInterruptor(t, ctx, conexao)
	return []deixado{pago, separando}
}

// expiracaoIgnoraOInterruptor roda dentro do binário com
// AZAMON_ENTREGA_SIMULACAO_ATIVA=false: um Pedido vencido com Reserva ATIVA
// ainda tem de expirar, liberando a unidade. A FR-34 é 🔒 e não tem
// interruptor — o da FR-33 é só da entrega simulada.
func expiracaoIgnoraOInterruptor(t *testing.T, ctx context.Context, conexao *pgx.Conn) {
	t.Helper()
	tx, err := conexao.Begin(ctx)
	if err != nil {
		t.Fatalf("abrir a transação: %v", err)
	}
	defer tx.Rollback(ctx)
	var pedidoID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO pedido.pedido (numero, comprador_id, status, subtotal_centavos, frete_centavos, total_centavos)
		VALUES ('AZ-INTERRUPTOR-0002', uuidv7(), 'AGUARDANDO_PAGAMENTO', 32900, 0, 32900)
		RETURNING id::text`).Scan(&pedidoID); err != nil {
		t.Fatalf("criar o Pedido: %v", err)
	}
	var produtoID string
	if err := tx.QueryRow(ctx, `
		SELECT id::text FROM catalogo.produto ORDER BY id LIMIT 1`).Scan(&produtoID); err != nil {
		t.Fatalf("escolher o Produto: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO catalogo.reserva_estoque (produto_id, pedido_id, quantidade, estado)
		VALUES ($1::uuid, $2::uuid, 1, 'ATIVA')`, produtoID, pedidoID); err != nil {
		t.Fatalf("criar a Reserva: %v", err)
	}
	// A linha do histórico por último, e no mesmo commit da Reserva: antes
	// dela o Pedido já seria candidato, e o tique de 50 ms o pegaria sem
	// Reserva para liberar.
	if _, err := tx.Exec(ctx, `
		INSERT INTO pedido.transicao_status (pedido_id, status_anterior, status_novo, ator, ocorrido_em)
		VALUES ($1::uuid, '', 'AGUARDANDO_PAGAMENTO', 'COMPRADOR', now() - interval '1 hour')`, pedidoID); err != nil {
		t.Fatalf("registrar o nascimento: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	prazo := time.Now().Add(30 * time.Second)
	for {
		var status, reserva string
		if err := conexao.QueryRow(ctx, `
			SELECT p.status, r.estado
			FROM pedido.pedido p JOIN catalogo.reserva_estoque r ON r.pedido_id = p.id
			WHERE p.id = $1::uuid`, pedidoID).Scan(&status, &reserva); err != nil {
			t.Fatalf("ler o Pedido e a Reserva: %v", err)
		}
		if status == "PAGAMENTO_RECUSADO" {
			if reserva != "LIBERADA" {
				t.Errorf("Reserva = %s com a simulação desligada; a expiração libera assim mesmo", reserva)
			}
			return
		}
		if time.Now().After(prazo) {
			t.Fatalf("status = %s com a simulação desligada; a expiração não tem interruptor", status)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// varreduraLevaOPedidoAPago é o comportamento-título da estória 1.7 medido no
// binário inteiro: sem isto, apagar o `go varrer(...)` de executar() ou trocar
// o caminho do webhook deixaria `go test ./...` verde.
//
// O Pedido, a Tentativa e a Reserva entram direto no banco — o que se exercita
// aqui é o relógio e o transporte, não a compra, que api/ já cobre. O total tem
// centavos em `,00`, que é a faixa que o Simulado aprova. A Reserva existe para
// a continuação: é ela que a simulação consolida em SEPARANDO → ENVIADO.
func varreduraLevaOPedidoAPago(t *testing.T, ctx context.Context, conexao *pgx.Conn) (string, string, int32) {
	t.Helper()
	var pedidoID string
	if err := conexao.QueryRow(ctx, `
		INSERT INTO pedido.pedido (numero, comprador_id, status, subtotal_centavos, frete_centavos, total_centavos)
		VALUES ('AZ-VARREDURA-000001', uuidv7(), 'AGUARDANDO_PAGAMENTO', 32900, 0, 32900)
		RETURNING id::text`).Scan(&pedidoID); err != nil {
		t.Fatalf("criar o Pedido: %v", err)
	}
	if _, err := conexao.Exec(ctx, `
		INSERT INTO pagamento.tentativa_pagamento (pedido_id, total_centavos, id_externo, numero)
		VALUES ($1::uuid, 32900, $2, 1)`, pedidoID, pagamento.IDExterno(pedidoID, 1)); err != nil {
		t.Fatalf("criar a Tentativa: %v", err)
	}
	var produtoID string
	var estoqueAntes int32
	if err := conexao.QueryRow(ctx, `
		SELECT id::text, estoque_total FROM catalogo.produto ORDER BY id LIMIT 1`).
		Scan(&produtoID, &estoqueAntes); err != nil {
		t.Fatalf("escolher o Produto: %v", err)
	}
	if _, err := conexao.Exec(ctx, `
		INSERT INTO catalogo.reserva_estoque (produto_id, pedido_id, quantidade, estado)
		VALUES ($1::uuid, $2::uuid, 1, 'ATIVA')`, produtoID, pedidoID); err != nil {
		t.Fatalf("criar a Reserva: %v", err)
	}

	// Mesmo estilo de espera do arranque: o tique é de 1 s e o atraso da
	// confirmação, de 100 ms.
	//
	// Quem responde é o histórico, e não o Status corrente: a simulação de
	// entrega roda no mesmo binário com intervalo de 200 ms, então o Pedido não
	// fica em PAGO — e uma espera que só quebrasse em `status == "PAGO"`
	// perderia a janela numa máquina carregada e acusaria regressão onde não há.
	prazo := time.Now().Add(30 * time.Second)
	for {
		var chegou bool
		if err := conexao.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pedido.transicao_status
				WHERE pedido_id = $1::uuid AND status_novo = 'PAGO'
			)`, pedidoID).Scan(&chegou); err != nil {
			t.Fatalf("ler o histórico: %v", err)
		}
		if chegou {
			break
		}
		if time.Now().After(prazo) {
			t.Fatal("o Pedido não registrou a passagem por PAGO; a varredura do binário não a aplicou")
		}
		time.Sleep(200 * time.Millisecond)
	}

	// E a confirmação atravessou a rota de verdade, com o segredo no
	// cabeçalho: uma linha na inbox, aplicada.
	var estado string
	if err := conexao.QueryRow(ctx, `
		SELECT c.estado FROM pagamento.confirmacao_recebida c
		JOIN pagamento.tentativa_pagamento t ON t.id = c.tentativa_id
		WHERE t.pedido_id = $1::uuid`, pedidoID).Scan(&estado); err != nil {
		t.Fatalf("ler a confirmação: %v", err)
	}
	if estado != "APLICADA" {
		t.Errorf("estado da confirmação = %s, quero APLICADA", estado)
	}
	return pedidoID, produtoID, estoqueAntes
}

// simulacaoLevaOPedidoAEntregue é o comportamento-título da estória 1.8 medido
// no binário inteiro. Sem isto, apagar o passo `simular` do tique deixaria
// `go test ./...` verde com o Pedido parado em PAGO para sempre — que é
// exatamente o fim que a épica existe para não ter.
//
// Ninguém empurra nada: o Pedido chega a PAGO pela varredura e daí em diante só
// o relógio o move, uma etapa por AZAMON_ENTREGA_INTERVALO vencido.
func simulacaoLevaOPedidoAEntregue(t *testing.T, ctx context.Context, conexao *pgx.Conn, pedidoID, produtoID string, estoqueAntes int32) {
	t.Helper()
	prazo := time.Now().Add(30 * time.Second)
	for {
		var status string
		if err := conexao.QueryRow(ctx,
			`SELECT status FROM pedido.pedido WHERE id = $1::uuid`, pedidoID).Scan(&status); err != nil {
			t.Fatalf("ler o Status: %v", err)
		}
		if status == "ENTREGUE" {
			break
		}
		if time.Now().After(prazo) {
			t.Fatalf("o Pedido ficou em %s; a simulação não o levou a ENTREGUE", status)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Uma linha de histórico por avanço, com a simulação como ator: PAGO →
	// SEPARANDO → ENVIADO → ENTREGUE são três, e nenhuma etapa foi pulada.
	var avancos int
	if err := conexao.QueryRow(ctx, `
		SELECT count(*) FROM pedido.transicao_status
		WHERE pedido_id = $1::uuid AND ator = 'SIMULACAO'`, pedidoID).Scan(&avancos); err != nil {
		t.Fatalf("contar as transições: %v", err)
	}
	if avancos != 3 {
		t.Errorf("transições da simulação = %d, quero 3", avancos)
	}

	// A consolidação aconteceu uma vez só: a Reserva saiu de ATIVA e o Estoque
	// total baixou exatamente a quantidade reservada — nem duas vezes, nem em
	// nenhuma outra transição.
	var reserva string
	if err := conexao.QueryRow(ctx, `
		SELECT estado FROM catalogo.reserva_estoque WHERE pedido_id = $1::uuid`, pedidoID).Scan(&reserva); err != nil {
		t.Fatalf("ler a Reserva: %v", err)
	}
	if reserva != "CONSOLIDADA" {
		t.Errorf("estado da Reserva = %s, quero CONSOLIDADA", reserva)
	}
	var estoqueDepois int32
	if err := conexao.QueryRow(ctx, `
		SELECT estoque_total FROM catalogo.produto WHERE id = $1::uuid`, produtoID).Scan(&estoqueDepois); err != nil {
		t.Fatalf("ler o Estoque: %v", err)
	}
	if estoqueDepois != estoqueAntes-1 {
		t.Errorf("estoque_total = %d, quero %d", estoqueDepois, estoqueAntes-1)
	}
}

// oTiqueExpiraNaOrdemDoAD6 e a FR-34 medida no binario inteiro: o passo
// `expirar` existe no tique, o prazo dele e o MESMO campo da Config de que a
// tela deriva `expira_em`, e uma aprovacao que chegou no prazo nao e descartada
// pelo relogio. Nada disso e alcancavel por api/, que chama os passos a mao e
// escolhe a ordem e o prazo que quiser.
//
// Tres Pedidos, todos com a inbox e o historico escritos numa transacao so —
// o binario ja esta ticando a cada segundo, e um Pedido que vira candidato
// antes de a confirmacao dele existir seria expirado no intervalo:
//   - `vencido`: historico de uma hora atras e nenhuma confirmacao. Expira, e
//     a unidade volta para a prateleira.
//   - `noPrazo`: historico de uma hora atras e uma aprovacao PENDENTE na inbox.
//     Termina PAGO. Invertidos `aplicar` e `expirar` no tique, ou tirada a
//     conferencia da inbox de dentro da trava, ele termina PAGAMENTO_RECUSADO.
//   - `recente`: historico de agora. Sobrevive a varios tiques e so entao
//     expira — e o que prende o prazo ao campo da Config, em vez de a qualquer
//     duracao que o tique quisesse usar.
func oTiqueExpiraNaOrdemDoAD6(t *testing.T, ctx context.Context, conexao *pgx.Conn) {
	t.Helper()
	// Tudo numa transacao: o Pedido so passa a ser candidato da expiracao
	// quando a linha do historico comita, e ate la a Tentativa e a confirmacao
	// dele ja estao no banco.
	nascer := func(numero, idade string, comConfirmacao bool) string {
		t.Helper()
		tx, err := conexao.Begin(ctx)
		if err != nil {
			t.Fatalf("abrir a transacao de %s: %v", numero, err)
		}
		defer tx.Rollback(ctx)
		var id string
		if err := tx.QueryRow(ctx, `
			INSERT INTO pedido.pedido (numero, comprador_id, status, subtotal_centavos, frete_centavos, total_centavos)
			VALUES ($1, uuidv7(), 'AGUARDANDO_PAGAMENTO', 32900, 0, 32900)
			RETURNING id::text`, numero).Scan(&id); err != nil {
			t.Fatalf("criar o Pedido %s: %v", numero, err)
		}
		if comConfirmacao {
			externo := pagamento.IDExterno(id, 1)
			if _, err := tx.Exec(ctx, `
				WITH t AS (
					INSERT INTO pagamento.tentativa_pagamento (pedido_id, total_centavos, id_externo, numero)
					VALUES ($1::uuid, 32900, $2, 1) RETURNING id
				)
				INSERT INTO pagamento.confirmacao_recebida (tentativa_id, chave_idempotencia, resultado, estado)
				SELECT t.id, $3, 'APROVADO', 'PENDENTE' FROM t`,
				id, externo, pagamento.ChaveIdempotencia(externo)); err != nil {
				t.Fatalf("criar a Tentativa e a confirmacao de %s: %v", numero, err)
			}
		}
		// A entrada em AGUARDANDO_PAGAMENTO: e dela, e so dela, que sai o
		// instante do vencimento — o mesmo que a tela mostra em `expira_em`.
		if _, err := tx.Exec(ctx, `
			INSERT INTO pedido.transicao_status (pedido_id, status_anterior, status_novo, ator, ocorrido_em)
			VALUES ($1::uuid, '', 'AGUARDANDO_PAGAMENTO', 'COMPRADOR', now() - $2::interval)`, id, idade); err != nil {
			t.Fatalf("registrar o nascimento de %s: %v", numero, err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit de %s: %v", numero, err)
		}
		return id
	}

	// A Reserva existe para a expiracao ter o que liberar (AD-3): o efeito
	// mora dentro de Transicionar, e o que a banca ve e a unidade de volta na
	// Pagina de Produto. Por isso a assercao e sobre o disponivel, e nao so
	// sobre o estado da linha.
	var produtoID string
	if err := conexao.QueryRow(ctx, `
		SELECT id::text FROM catalogo.produto ORDER BY id LIMIT 1`).Scan(&produtoID); err != nil {
		t.Fatalf("escolher o Produto: %v", err)
	}
	disponivel := func() int32 {
		t.Helper()
		var n int32
		if err := conexao.QueryRow(ctx, `
			SELECT estoque_disponivel FROM catalogo.produto_visivel WHERE id = $1::uuid`, produtoID).Scan(&n); err != nil {
			t.Fatalf("ler o Estoque disponivel: %v", err)
		}
		return n
	}

	vencido := nascer("AZ-EXPIRACAO-00001", "1 hour", false)
	if _, err := conexao.Exec(ctx, `
		INSERT INTO catalogo.reserva_estoque (produto_id, pedido_id, quantidade, estado)
		VALUES ($1::uuid, $2::uuid, 1, 'ATIVA')`, produtoID, vencido); err != nil {
		t.Fatalf("criar a Reserva: %v", err)
	}
	comReserva := disponivel()
	noPrazo := nascer("AZ-EXPIRACAO-00002", "1 hour", true)
	recente := nascer("AZ-EXPIRACAO-00003", "0", false)

	status := func(id string) string {
		t.Helper()
		var s string
		if err := conexao.QueryRow(ctx,
			`SELECT status FROM pedido.pedido WHERE id = $1::uuid`, id).Scan(&s); err != nil {
			t.Fatalf("ler o Status: %v", err)
		}
		return s
	}
	esperarSair := func(id, nome string) string {
		t.Helper()
		prazo := time.Now().Add(30 * time.Second)
		for {
			if s := status(id); s != "AGUARDANDO_PAGAMENTO" {
				return s
			}
			if time.Now().After(prazo) {
				t.Fatalf("o tique nao resolveu o Pedido %s", nome)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}

	if s := esperarSair(vencido, "vencido"); s != "PAGAMENTO_RECUSADO" {
		t.Errorf("o vencido esta em %s; quero PAGAMENTO_RECUSADO", s)
	}
	var motivo, estado string
	if err := conexao.QueryRow(ctx, `
		SELECT coalesce(t.motivo, ''), r.estado
		FROM pedido.transicao_status t, catalogo.reserva_estoque r
		WHERE t.pedido_id = $1::uuid AND r.pedido_id = $1::uuid
		ORDER BY t.ocorrido_em DESC, t.id DESC LIMIT 1`, vencido).Scan(&motivo, &estado); err != nil {
		t.Fatalf("ler a transicao e a Reserva: %v", err)
	}
	if motivo != "TEMPO_ESGOTADO" || estado != "LIBERADA" {
		t.Errorf("motivo = %q e Reserva = %s; quero TEMPO_ESGOTADO e LIBERADA", motivo, estado)
	}
	// A prateleira: sem isto, uma expiracao que gravasse o historico e deixasse
	// a Reserva presa passaria por aqui.
	if agora := disponivel(); agora != comReserva+1 {
		t.Errorf("Estoque disponivel = %d, quero %d: a expiracao devolve a unidade", agora, comReserva+1)
	}

	// O que a ordem do tique e a conferencia da inbox protegem juntas. Quem
	// responde e o historico, e nao o Status corrente: a simulacao de entrega
	// roda neste binario a 200 ms e pode ja ter levado o Pedido adiante de
	// PAGO quando a leitura chega. A primeira saida de AGUARDANDO_PAGAMENTO e
	// o que o tique decidiu, e nenhuma linha pode ter ido a
	// PAGAMENTO_RECUSADO — nem antes, nem depois.
	esperarSair(noPrazo, "pago no prazo")
	var primeiraSaida string
	var recusas int
	if err := conexao.QueryRow(ctx, `
		SELECT (SELECT status_novo FROM pedido.transicao_status
		        WHERE pedido_id = $1::uuid AND status_anterior = 'AGUARDANDO_PAGAMENTO'
		        ORDER BY ocorrido_em, id LIMIT 1),
		       (SELECT count(*) FROM pedido.transicao_status
		        WHERE pedido_id = $1::uuid AND status_novo = 'PAGAMENTO_RECUSADO')`,
		noPrazo).Scan(&primeiraSaida, &recusas); err != nil {
		t.Fatalf("ler o historico do Pedido pago no prazo: %v", err)
	}
	if primeiraSaida != "PAGO" || recusas != 0 {
		t.Errorf("o Pedido pago no prazo saiu de AGUARDANDO_PAGAMENTO para %s, com %d linha(s) em PAGAMENTO_RECUSADO; quero PAGO e nenhuma — `aplicar` corre antes de `expirar`, e a inbox e conferida sob a trava", primeiraSaida, recusas)
	}

	// `recente` nasceu agora, com o prazo da Config bem acima do tique: varios
	// tiques tem de passar por ele sem o mover. Trocado o prazo por qualquer
	// outra duracao curta, ou pelo intervalo da entrega, esta espera falha.
	time.Sleep(expiracaoDeTesteDoBinario / 2)
	if s := status(recente); s != "AGUARDANDO_PAGAMENTO" {
		t.Fatalf("o Pedido recente esta em %s antes de metade do prazo; o tique nao usa AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO", s)
	}
	if s := esperarSair(recente, "recente"); s != "PAGAMENTO_RECUSADO" {
		t.Errorf("o Pedido recente esta em %s depois do prazo; quero PAGAMENTO_RECUSADO", s)
	}
}
