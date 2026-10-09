//go:build ignore

// Comando gerar escreve o SQL do Catálogo Semeado em db/semente/ a partir da
// lista escrita à mão logo abaixo. Rode na raiz do repositório:
//
//	go run media/gerar.go
//
// Gerador e saída ficam os dois versionados de propósito. O gerador não
// desenha imagem nem acessa a rede: a foto de cada Produto já está em
// media/{apelido}.webp, e ele só confere que ela existe (AD-12). Antes de
// escrever qualquer coisa, ele falha com nome ou apelido repetido e com imagem
// ausente — saída parcial seria um SQL apontando para arquivo que não existe.
//
// Os identificadores são derivados por SHA-1 do nome (o formato do uuid v5),
// e não sorteados: `DEFAULT uuidv7()` é o certo para dado nascido em execução
// e o errado para a semente, onde cada `down -v` daria identificador novo.
package main

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
)

// ---------------------------------------------------------------------------
// A lista, escrita à mão. São os 194 Produtos do DummyJSON (Estória 8.1),
// traduzidos e sem marca: o PRD pede Produtos fictícios, e o nome descreve o
// objeto ("Fone Over-ear Prata"). `origem` é o id do Produto no DummyJSON, e a
// foto em media/ é o images[0] dele, como veio.
//
// O preço é escolhido, não só convertido: reais = max(1, round(USD × 5,50)),
// sempre em `,00`, porque o Provedor Simulado decide pelos centavos do total e
// o Frete é em reais inteiros. As duas exceções são de propósito: o
// Carregador (`,90`) é o Produto que recusa, e a Bola de Futebol (`,95`), o
// que expira.
// ---------------------------------------------------------------------------

type produto struct {
	origem    int
	nome      string
	descricao string
	centavos  int64
	categoria string
	vendedor  string
}

var vendedores = []string{
	"Atlântico Importados",
	"Vale do Sol Distribuidora",
	"Sertão Empório",
	"Casa Boa Utilidades",
	"Pampa Esportes",
}

var categorias = []string{
	"Eletrônicos",
	"Casa e Cozinha",
	"Moda",
	"Mercado",
	"Esporte e Lazer",
	"Relógios e Joias",
	"Beleza e Perfumaria",
	"Veículos",
}

var produtos = []produto{
	// Eletrônicos (38)
	{78, "Notebook 14 Polegadas Cinza-Espacial", "Notebook fino e potente, com processador de alto desempenho e tela de alta resolução.", 1100000, "Eletrônicos", "Atlântico Importados"},
	{79, "Notebook com Tela Dupla", "Notebook de alto desempenho com uma segunda tela sobre o teclado, para quem cria e produz.", 990000, "Eletrônicos", "Atlântico Importados"},
	{80, "Notebook Ultrafino com Tela de Toque", "Notebook fino e elegante, com tela sensível ao toque de alta resolução, para levar a qualquer lugar.", 770000, "Eletrônicos", "Atlântico Importados"},
	{81, "Notebook Conversível 2 em 1", "Notebook com dobradiça flexível que vira tablet, versátil e portátil.", 605000, "Eletrônicos", "Atlântico Importados"},
	{82, "Notebook Compacto 13 Polegadas", "Notebook compacto e potente, com tela de bordas mínimas.", 825000, "Eletrônicos", "Atlântico Importados"},
	{99, "Caixa de Som Inteligente com Assistente", "Caixa de som com controle por voz, som de qualidade e central para a casa conectada.", 55000, "Eletrônicos", "Atlântico Importados"},
	{100, "Fone sem Fio com Estojo", "Fones sem fio de pareamento fácil e som de qualidade, com estojo de carga.", 71500, "Eletrônicos", "Atlântico Importados"},
	{101, "Fone Over-ear Prata", "Fone over-ear com áudio de alta fidelidade, equalização adaptativa e cancelamento ativo de ruído.", 302500, "Eletrônicos", "Atlântico Importados"},
	{102, "Base de Carregamento sem Fio", "Base de carga por indução: basta apoiar o aparelho compatível sobre ela.", 44000, "Eletrônicos", "Atlântico Importados"},
	{103, "Caixa de Som Inteligente Compacta", "Caixa de som esférica e compacta, com áudio surpreendente para a casa conectada.", 55000, "Eletrônicos", "Atlântico Importados"},
	{104, "Carregador de Celular com Cabo", "Carregador de tomada com cabo USB, para carga rápida e eficiente do celular.", 11090, "Eletrônicos", "Atlântico Importados"},
	{105, "Bateria Magnética para Celular", "Bateria portátil que se prende por ímã ao celular compatível e estende a carga.", 55000, "Eletrônicos", "Atlântico Importados"},
	{106, "Relógio Inteligente Dourado", "Relógio inteligente com medição de batimentos, registro de atividades e tela nítida.", 192500, "Eletrônicos", "Atlântico Importados"},
	{107, "Fone sem Fio com Cordão", "Fone intra-auricular sem fio, com pontas magnéticas e até 12 horas de bateria.", 27500, "Eletrônicos", "Atlântico Importados"},
	{108, "Capa de Silicone Ameixa", "Capa de silicone protetora, com ímã para acessórios magnéticos, na cor ameixa.", 16500, "Eletrônicos", "Atlântico Importados"},
	{109, "Tripé de Mesa para Celular", "Suporte ajustável para fotos e vídeos estáveis com o celular.", 11000, "Eletrônicos", "Atlântico Importados"},
	{110, "Iluminador de Anel com Tripé", "Luz de LED em anel com tripé ajustável, para selfies e chamadas de vídeo bem iluminadas.", 8200, "Eletrônicos", "Atlântico Importados"},
	{111, "Bastão de Selfie Retrátil", "Bastão extensível e dobrável para selfies e fotos em grupo, compatível com celulares.", 7100, "Eletrônicos", "Atlântico Importados"},
	{112, "Tripé Profissional para Câmera", "Tripé de estúdio para movimentos de câmera suaves e precisos, em gravação e transmissão.", 275000, "Eletrônicos", "Atlântico Importados"},
	{121, "Smartphone Compacto Grafite", "Smartphone de tela de 4 polegadas, compacto e confiável para o dia a dia.", 110000, "Eletrônicos", "Atlântico Importados"},
	{122, "Smartphone Prata de Tela Ampliada", "Smartphone de tela maior e desempenho melhorado, fino e capaz.", 165000, "Eletrônicos", "Atlântico Importados"},
	{123, "Smartphone Azul com Câmera Tripla", "Smartphone de ponta com câmera tripla, processador potente e tela de alta qualidade.", 605000, "Eletrônicos", "Atlântico Importados"},
	{124, "Smartphone Preto com Tela OLED", "Smartphone com tela OLED sem bordas, reconhecimento facial e ótimo desempenho.", 495000, "Eletrônicos", "Atlântico Importados"},
	{125, "Smartphone Dourado Intermediário", "Smartphone intermediário de desenho fino, com bom equilíbrio entre desempenho e preço.", 137500, "Eletrônicos", "Atlântico Importados"},
	{126, "Smartphone Prata com Câmera Quádrupla", "Smartphone focado em fotografia, com câmera quádrupla e bom desempenho.", 220000, "Eletrônicos", "Atlântico Importados"},
	{127, "Smartphone Degradê Azul e Roxo", "Smartphone de visual marcante, com câmera dupla e desempenho confiável.", 165000, "Eletrônicos", "Atlântico Importados"},
	{128, "Smartphone Verde-Claro Básico", "Smartphone de entrada com o essencial para o dia a dia e uso descomplicado.", 82500, "Eletrônicos", "Atlântico Importados"},
	{129, "Smartphone Roxo com Câmera Retrátil", "Smartphone com câmera frontal retrátil, tela ampla e bom equilíbrio entre desempenho e foto.", 165000, "Eletrônicos", "Atlântico Importados"},
	{130, "Smartphone Azul com Câmera de Alta Resolução", "Smartphone com câmera quádrupla de alta resolução, para fotos e vídeos detalhados.", 192500, "Eletrônicos", "Atlântico Importados"},
	{131, "Smartphone Preto Clássico", "Smartphone de linhas elegantes, com tela de alta resolução e câmera potente.", 165000, "Eletrônicos", "Atlântico Importados"},
	{132, "Smartphone Preto com Tela Infinita", "Smartphone com tela de bordas curvas, câmera avançada e muita tecnologia.", 275000, "Eletrônicos", "Atlântico Importados"},
	{133, "Smartphone Grafite com Tela AMOLED", "Smartphone topo de linha com tela AMOLED dinâmica, câmera versátil e desempenho potente.", 385000, "Eletrônicos", "Atlântico Importados"},
	{134, "Smartphone Azul-Gelo", "Smartphone intermediário estiloso, com tela vibrante e câmera competente.", 137500, "Eletrônicos", "Atlântico Importados"},
	{135, "Smartphone Dourado para Selfies", "Smartphone com câmera dupla e foco em selfies de qualidade, com tela de entalhe.", 165000, "Eletrônicos", "Atlântico Importados"},
	{136, "Smartphone Vermelho com Leitor na Tela", "Smartphone com leitor de digital sob a tela, tela de alta resolução e câmera avançada.", 275000, "Eletrônicos", "Atlântico Importados"},
	{159, "Tablet Compacto 8 Polegadas", "Tablet compacto e potente, com tela nítida e desenho fino.", 275000, "Eletrônicos", "Atlântico Importados"},
	{160, "Tablet Grafite com Caneta", "Tablet de alto desempenho com tela AMOLED grande e caneta, para trabalho e lazer.", 330000, "Eletrônicos", "Atlântico Importados"},
	{161, "Tablet Branco 10 Polegadas", "Tablet versátil com tela vibrante e bateria de longa duração.", 192500, "Eletrônicos", "Atlântico Importados"},

	// Casa e Cozinha (40)
	{11, "Cama de Casal em Madeira", "Cama de casal com cabeceira torneada em madeira, para um quarto confortável e elegante.", 1045000, "Casa e Cozinha", "Casa Boa Utilidades"},
	{12, "Sofá de Três Lugares", "Sofá com base de madeira e estofado claro, confortável e sofisticado para a sala.", 1375000, "Casa e Cozinha", "Casa Boa Utilidades"},
	{13, "Criado-Mudo em Cerejeira", "Criado-mudo em madeira escura, com gaveta e nicho, prático ao lado da cama.", 165000, "Casa e Cozinha", "Casa Boa Utilidades"},
	{14, "Cadeira Executiva Giratória", "Cadeira de escritório ergonômica, com assento envolvente e base giratória com rodízios.", 275000, "Casa e Cozinha", "Casa Boa Utilidades"},
	{15, "Gabinete de Banheiro com Espelho", "Gabinete de madeira com cuba e espelho combinando, para um banheiro com personalidade.", 440000, "Casa e Cozinha", "Casa Boa Utilidades"},
	{43, "Balanço Decorativo", "Peça de decoração cheia de detalhes, que traz charme e leveza a qualquer ambiente.", 33000, "Casa e Cozinha", "Casa Boa Utilidades"},
	{44, "Porta-Retratos Árvore Genealógica", "Porta-retratos com vários espaços para fotos, que conta a história da família.", 16500, "Casa e Cozinha", "Casa Boa Utilidades"},
	{45, "Planta Artificial Decorativa", "Planta artificial que traz verde para a casa sem precisar de cuidados.", 22000, "Casa e Cozinha", "Casa Boa Utilidades"},
	{46, "Vaso para Plantas", "Vaso de linhas modernas para plantas de interior ou de varanda.", 8200, "Casa e Cozinha", "Casa Boa Utilidades"},
	{47, "Abajur de Mesa", "Abajur decorativo de desenho moderno, para luz ambiente ou de leitura.", 27500, "Casa e Cozinha", "Casa Boa Utilidades"},
	{48, "Espátula de Bambu", "Espátula de bambu para virar, mexer e servir, feita de material renovável.", 4400, "Casa e Cozinha", "Casa Boa Utilidades"},
	{49, "Copo de Alumínio Preto", "Copo de alumínio resistente, para bebidas quentes ou geladas.", 3300, "Casa e Cozinha", "Casa Boa Utilidades"},
	{50, "Batedor de Arame Preto", "Batedor de cabo ergonômico para bater claras, massas e molhos.", 5500, "Casa e Cozinha", "Casa Boa Utilidades"},
	{51, "Liquidificador Compacto", "Liquidificador potente e compacto para vitaminas, batidas e muito mais.", 22000, "Casa e Cozinha", "Casa Boa Utilidades"},
	{52, "Wok de Aço Carbono", "Panela funda para saltear, refogar e fritar, com distribuição de calor uniforme.", 16500, "Casa e Cozinha", "Casa Boa Utilidades"},
	{53, "Tábua de Corte", "Tábua resistente e higiênica para cortar e picar alimentos.", 7100, "Casa e Cozinha", "Casa Boa Utilidades"},
	{54, "Espremedor de Frutas Cítricas", "Espremedor amarelo, prático para tirar o suco de limões e laranjas.", 4900, "Casa e Cozinha", "Casa Boa Utilidades"},
	{55, "Fatiador de Ovos", "Fatia ovos cozidos por igual, para saladas e sanduíches.", 3800, "Casa e Cozinha", "Casa Boa Utilidades"},
	{56, "Fogão Elétrico Portátil", "Fogareiro elétrico prático e eficiente, bom para cozinhas pequenas ou como boca extra.", 27500, "Casa e Cozinha", "Casa Boa Utilidades"},
	{57, "Peneira de Malha Fina", "Peneira para coar líquidos e peneirar farinhas, com malha bem fina.", 5500, "Casa e Cozinha", "Casa Boa Utilidades"},
	{58, "Garfo de Mesa", "Garfo de desenho ergonômico e resistente, para o uso diário.", 2200, "Casa e Cozinha", "Casa Boa Utilidades"},
	{59, "Copo de Vidro", "Copo de vidro transparente e elegante, para todo tipo de bebida.", 2700, "Casa e Cozinha", "Casa Boa Utilidades"},
	{60, "Ralador Preto", "Ralador de lâminas afiadas para queijos, legumes e mais.", 6000, "Casa e Cozinha", "Casa Boa Utilidades"},
	{61, "Mixer de Mão", "Mixer compacto e potente para bater, triturar e fazer purês.", 19200, "Casa e Cozinha", "Casa Boa Utilidades"},
	{62, "Forma de Gelo", "Forma prática para fazer cubos de gelo e manter as bebidas geladas.", 3300, "Casa e Cozinha", "Casa Boa Utilidades"},
	{63, "Coador de Cozinha", "Coador de malha fina para peneirar e escorrer ingredientes secos e úmidos.", 4400, "Casa e Cozinha", "Casa Boa Utilidades"},
	{64, "Faca de Cozinha", "Faca de lâmina afiada e cabo ergonômico para picar, fatiar e cortar.", 8200, "Casa e Cozinha", "Casa Boa Utilidades"},
	{65, "Marmita com Divisórias", "Marmita portátil com compartimentos, para levar a refeição para qualquer lugar.", 7100, "Casa e Cozinha", "Casa Boa Utilidades"},
	{66, "Forno de Micro-ondas", "Micro-ondas compacto para cozinhar, esquentar e descongelar com rapidez.", 49500, "Casa e Cozinha", "Casa Boa Utilidades"},
	{67, "Suporte para Canecas", "Suporte em forma de árvore que organiza as canecas e economiza espaço.", 8800, "Casa e Cozinha", "Casa Boa Utilidades"},
	{68, "Frigideira Antiaderente", "Frigideira com revestimento antiaderente para fritar e refogar, fácil de limpar.", 13700, "Casa e Cozinha", "Casa Boa Utilidades"},
	{69, "Prato Raso", "Prato resistente e bonito para as refeições do dia a dia ou ocasiões especiais.", 2200, "Casa e Cozinha", "Casa Boa Utilidades"},
	{70, "Pegador Vermelho", "Pegador de cozinha versátil para preparar e servir.", 3800, "Casa e Cozinha", "Casa Boa Utilidades"},
	{71, "Panela Inox com Tampa de Vidro", "Panela para ferver e cozinhar, com tampa de vidro para acompanhar o preparo.", 22000, "Casa e Cozinha", "Casa Boa Utilidades"},
	{72, "Espátula Vazada", "Espátula com fendas para virar alimentos e escorrer o excesso de líquido.", 4900, "Casa e Cozinha", "Casa Boa Utilidades"},
	{73, "Porta-Temperos", "Organizador para temperos e condimentos, sempre à mão e em ordem.", 11000, "Casa e Cozinha", "Casa Boa Utilidades"},
	{74, "Colher de Cozinha", "Colher ergonômica e resistente para mexer, servir e provar.", 2700, "Casa e Cozinha", "Casa Boa Utilidades"},
	{75, "Bandeja de Servir", "Bandeja decorativa para servir petiscos, aperitivos e bebidas.", 9300, "Casa e Cozinha", "Casa Boa Utilidades"},
	{76, "Rolo de Massa de Madeira", "Rolo de madeira com cabos firmes para abrir massas de espessura uniforme.", 6600, "Casa e Cozinha", "Casa Boa Utilidades"},
	{77, "Descascador Amarelo", "Descascador prático para frutas e legumes.", 3300, "Casa e Cozinha", "Casa Boa Utilidades"},

	// Moda (35)
	{83, "Camisa Xadrez Azul e Preta", "Camisa masculina de manga longa em xadrez clássico, para ocasiões casuais e semiformais.", 16500, "Moda", "Vale do Sol Distribuidora"},
	{84, "Camiseta Branca Estampada", "Camiseta masculina de algodão com estampa gráfica, casual e confortável.", 13700, "Moda", "Vale do Sol Distribuidora"},
	{85, "Camisa Xadrez Vermelha", "Camisa masculina de flanela em xadrez vermelho e preto, curinga no guarda-roupa.", 19200, "Moda", "Vale do Sol Distribuidora"},
	{86, "Camisa Floral de Manga Curta", "Camisa masculina leve com estampa floral, para os dias quentes.", 11000, "Moda", "Vale do Sol Distribuidora"},
	{87, "Camisa Xadrez Verde", "Camisa masculina de manga longa em xadrez, que dá um toque arrumado ao visual.", 15400, "Moda", "Vale do Sol Distribuidora"},
	{88, "Tênis de Cano Alto Vermelho e Branco", "Tênis de basquete de cano alto, de estilo marcante e bom desempenho.", 82500, "Moda", "Vale do Sol Distribuidora"},
	{89, "Chuteira de Beisebol Branca e Verde", "Chuteira com travas para aderência máxima no campo de beisebol.", 44000, "Moda", "Vale do Sol Distribuidora"},
	{90, "Tênis Retrô Colorido", "Tênis casual que mistura estilo retrô e conforto moderno.", 49500, "Moda", "Vale do Sol Distribuidora"},
	{91, "Tênis Branco com Seta Vermelha", "Tênis esportivo branco com detalhe vermelho, estiloso e funcional.", 66000, "Moda", "Vale do Sol Distribuidora"},
	{92, "Tênis Branco e Vermelho", "Tênis casual branco com solado vermelho, confortável para o dia a dia.", 60500, "Moda", "Vale do Sol Distribuidora"},
	{154, "Óculos de Sol Tartaruga", "Óculos de armação clássica, com lentes escuras e proteção UV.", 16500, "Moda", "Vale do Sol Distribuidora"},
	{155, "Óculos de Sol Aviador", "Óculos aviador com lentes degradê e proteção UV, para qualquer ocasião.", 13700, "Moda", "Vale do Sol Distribuidora"},
	{156, "Óculos de Sol Verde e Preto", "Óculos com lentes verdes e armação preta, que chamam atenção.", 19200, "Moda", "Vale do Sol Distribuidora"},
	{157, "Óculos de Festa Pixelado", "Óculos divertidos para dar um toque brincalhão à fantasia ou à festa.", 11000, "Moda", "Vale do Sol Distribuidora"},
	{158, "Óculos de Sol Transparente", "Óculos de armação transparente e desenho simples, com proteção UV.", 12600, "Moda", "Vale do Sol Distribuidora"},
	{162, "Vestido Rodado Azul", "Vestido rodado azul com estampa de círculos, confortável e cheio de charme.", 16500, "Moda", "Vale do Sol Distribuidora"},
	{163, "Vestido Longo Estampado de Folhas", "Vestido leve de alças, com estampa de folhas, para os dias de calor.", 11000, "Moda", "Vale do Sol Distribuidora"},
	{164, "Vestido Longo Cinza", "Vestido cinza com botões, curinga para ocasiões variadas.", 19200, "Moda", "Vale do Sol Distribuidora"},
	{165, "Vestido Curto Verde-Água", "Vestido curto com babados e laço na cintura, para passeios ou ocasiões especiais.", 13700, "Moda", "Vale do Sol Distribuidora"},
	{166, "Vestido Xadrez com Laço", "Vestido xadrez com laço preto, de charme tradicional para o outono e o inverno.", 22000, "Moda", "Vale do Sol Distribuidora"},
	{172, "Bolsa de Mão Azul", "Bolsa espaçosa, com vários compartimentos, em azul intenso.", 27500, "Moda", "Vale do Sol Distribuidora"},
	{173, "Bolsa de Couro Caramelo", "Bolsa de couro de alta qualidade, de desenho atemporal e acabamento durável.", 71500, "Moda", "Vale do Sol Distribuidora"},
	{174, "Bolsa Estruturada Azul-Claro", "Bolsa estruturada de alças curtas, elegante e sofisticada.", 330000, "Moda", "Vale do Sol Distribuidora"},
	{175, "Mochila de Couro Sintético Branca", "Mochila feminina branca, prática e com bom espaço interno.", 22000, "Moda", "Vale do Sol Distribuidora"},
	{176, "Bolsa de Mão Preta", "Bolsa preta clássica e versátil, que combina com tudo.", 33000, "Moda", "Vale do Sol Distribuidora"},
	{177, "Vestido de Festa Floral", "Vestido longo de noite, com estampa floral sobre fundo escuro, para eventos formais.", 71500, "Moda", "Vale do Sol Distribuidora"},
	{178, "Corpete de Couro com Saia Vermelha", "Conjunto de corpete e saia longa em vermelho, ousado e marcante.", 49500, "Moda", "Vale do Sol Distribuidora"},
	{179, "Corpete com Saia Preta", "Conjunto de corpete e saia longa preta, elegante e coordenado.", 44000, "Moda", "Vale do Sol Distribuidora"},
	{180, "Vestido Branco de Poá", "Vestido rodado com estampa de bolinhas, leve e divertido para o dia a dia.", 27500, "Moda", "Vale do Sol Distribuidora"},
	{181, "Vestido Bordô de Manga Longa", "Vestido de manga longa em tons de vinho e rosa, moderno e marcante.", 99000, "Moda", "Vale do Sol Distribuidora"},
	{185, "Scarpin Preto e Marrom", "Sapato de salto alto preto, confortável e sofisticado.", 11000, "Moda", "Vale do Sol Distribuidora"},
	{186, "Sandália Preta de Tira Larga", "Sandália elegante e sofisticada, para ocasiões formais.", 44000, "Moda", "Vale do Sol Distribuidora"},
	{187, "Scarpin Dourado", "Sapato de salto em tom dourado, glamoroso para ocasiões especiais.", 27500, "Moda", "Vale do Sol Distribuidora"},
	{188, "Scarpin Preto Clássico", "Sapato de salto versátil e confortável, para o dia a dia.", 16500, "Moda", "Vale do Sol Distribuidora"},
	{189, "Scarpin Vermelho", "Sapato de salto vermelho vibrante, para festa ou passeio.", 19200, "Moda", "Vale do Sol Distribuidora"},

	// Mercado (27)
	{16, "Maçã Vermelha", "Maçãs frescas e crocantes, boas para o lanche ou para receitas doces e salgadas.", 1100, "Mercado", "Sertão Empório"},
	{17, "Bife de Contrafilé", "Corte bovino de qualidade, bom para grelhar ou preparar no ponto que preferir.", 7100, "Mercado", "Sertão Empório"},
	{18, "Ração para Gatos", "Ração completa, formulada para as necessidades nutricionais do seu gato.", 4900, "Mercado", "Sertão Empório"},
	{19, "Peito de Frango", "Carne de frango fresca e macia, para as mais variadas receitas.", 5500, "Mercado", "Sertão Empório"},
	{20, "Óleo de Cozinha", "Óleo versátil para fritar, refogar e preparar o dia a dia na cozinha.", 2700, "Mercado", "Sertão Empório"},
	{21, "Pepino Japonês", "Pepinos crocantes e refrescantes, ótimos em saladas e como acompanhamento.", 800, "Mercado", "Sertão Empório"},
	{22, "Ração para Cães", "Ração formulada para dar ao seu cachorro os nutrientes essenciais.", 6000, "Mercado", "Sertão Empório"},
	{23, "Ovos Brancos", "Ovos frescos, ingrediente versátil para bolos, receitas e o café da manhã.", 1600, "Mercado", "Sertão Empório"},
	{24, "Posta de Peixe", "Posta de peixe de qualidade, para grelhar, assar ou selar na frigideira.", 8200, "Mercado", "Sertão Empório"},
	{25, "Pimentão Verde", "Pimentão fresco e vistoso, para dar cor e sabor aos seus pratos.", 700, "Mercado", "Sertão Empório"},
	{26, "Pimenta Verde", "Pimenta ardida, para dar picância às suas receitas favoritas.", 500, "Mercado", "Sertão Empório"},
	{27, "Mel Puro em Pote", "Mel puro e natural em pote prático, para adoçar bebidas ou regar sobremesas.", 3800, "Mercado", "Sertão Empório"},
	{28, "Pote de Sorvete", "Sorvete cremoso, em vários sabores, para uma sobremesa gelada.", 3000, "Mercado", "Sertão Empório"},
	{29, "Suco de Fruta", "Suco de fruta refrescante e rico em vitaminas.", 2200, "Mercado", "Sertão Empório"},
	{30, "Kiwi", "Kiwis nutritivos, bons para o lanche ou para um toque tropical nos pratos.", 1400, "Mercado", "Sertão Empório"},
	{31, "Limão Siciliano", "Limões perfumados e azedinhos, para cozinhar, confeitar ou preparar bebidas.", 400, "Mercado", "Sertão Empório"},
	{32, "Leite Integral", "Leite fresco e nutritivo, básico para receitas e para o consumo diário.", 1900, "Mercado", "Sertão Empório"},
	{33, "Amora", "Amoras doces e suculentas, para comer ao natural ou com sobremesas e cereais.", 2700, "Mercado", "Sertão Empório"},
	{34, "Café Solúvel", "Café solúvel de sabor encorpado, para uma xícara rápida a qualquer hora.", 4400, "Mercado", "Sertão Empório"},
	{35, "Batata Inglesa", "Batatas versáteis, ótimas assadas, em purê ou como acompanhamento.", 1300, "Mercado", "Sertão Empório"},
	{36, "Proteína em Pó", "Suplemento em pó para complementar a dieta com proteína.", 11000, "Mercado", "Sertão Empório"},
	{37, "Cebola Roxa", "Cebolas roxas aromáticas, que dão profundidade aos pratos salgados.", 1100, "Mercado", "Sertão Empório"},
	{38, "Arroz Branco", "Arroz de qualidade, base versátil para muitos pratos.", 3300, "Mercado", "Sertão Empório"},
	{39, "Refrigerantes Sortidos", "Refrigerantes em sabores variados, para servir gelado.", 1100, "Mercado", "Sertão Empório"},
	{40, "Morango", "Morangos doces e suculentos, para lanches, sobremesas e vitaminas.", 2200, "Mercado", "Sertão Empório"},
	{41, "Caixa de Lenços de Papel", "Lenços de papel macios e absorventes, em caixa prática para o dia a dia.", 1400, "Mercado", "Sertão Empório"},
	{42, "Água Mineral", "Água mineral pura e refrescante, para se manter hidratado o dia todo.", 500, "Mercado", "Sertão Empório"},

	// Esporte e Lazer (17)
	{137, "Bola de Futebol Americano", "Bola oval clássica do futebol americano, feita para lançar e receber.", 11000, "Esporte e Lazer", "Pampa Esportes"},
	{138, "Bola de Beisebol", "Bola de beisebol com capa de couro resistente, para arremessar, rebater e defender.", 4900, "Esporte e Lazer", "Pampa Esportes"},
	{139, "Luva de Beisebol", "Luva de proteção para pegar e defender a bola, com conforto e controle.", 13700, "Esporte e Lazer", "Pampa Esportes"},
	{140, "Bola de Basquete", "Bola para quadras internas e externas, feita para driblar, arremessar e passar.", 8200, "Esporte e Lazer", "Pampa Esportes"},
	{141, "Aro de Basquete com Rede", "Aro resistente com rede, para fixar na tabela e treinar os arremessos.", 22000, "Esporte e Lazer", "Pampa Esportes"},
	{142, "Bola de Críquete", "Bola de couro dura, com costura saliente, usada no críquete.", 7100, "Esporte e Lazer", "Pampa Esportes"},
	{143, "Taco de Críquete", "Taco de madeira para rebater a bola no críquete.", 16500, "Esporte e Lazer", "Pampa Esportes"},
	{144, "Capacete de Críquete", "Capacete de proteção para batedores e guardiões contra arremessos rápidos.", 24700, "Esporte e Lazer", "Pampa Esportes"},
	{145, "Wicket de Críquete", "Conjunto de três estacas e duas travessas que formam o wicket do críquete.", 16500, "Esporte e Lazer", "Pampa Esportes"},
	{146, "Peteca de Badminton", "Peteca com penas naturais, estável e precisa em jogadas rápidas.", 3300, "Esporte e Lazer", "Pampa Esportes"},
	{147, "Bola de Futebol", "Bola de futebol de campo em tamanho oficial, feita para chutar e passar.", 9995, "Esporte e Lazer", "Pampa Esportes"},
	{148, "Bola de Golfe", "Bola de golfe com covinhas que dão sustentação e distância à tacada.", 5500, "Esporte e Lazer", "Pampa Esportes"},
	{149, "Taco de Golfe de Ferro", "Taco de cabeça metálica para aproximações e tacadas curtas.", 27500, "Esporte e Lazer", "Pampa Esportes"},
	{150, "Taco de Beisebol de Alumínio", "Taco leve e resistente de liga metálica, para jogo e treino de rebatidas.", 16500, "Esporte e Lazer", "Pampa Esportes"},
	{151, "Bola de Tênis", "Bola de tênis para partidas e treinos.", 3800, "Esporte e Lazer", "Pampa Esportes"},
	{152, "Raquete de Tênis", "Raquete com encordoamento firme e empunhadura confortável.", 27500, "Esporte e Lazer", "Pampa Esportes"},
	{153, "Bola de Vôlei", "Bola de vôlei para passar, levantar e atacar por cima da rede.", 6600, "Esporte e Lazer", "Pampa Esportes"},

	// Relógios e Joias (14)
	{93, "Relógio de Couro Marrom", "Relógio de desenho clássico, com pulseira de couro legítimo e mostrador branco.", 49500, "Relógios e Joias", "Vale do Sol Distribuidora"},
	{94, "Relógio de Aço com Cronógrafo", "Relógio elegante e preciso, com pulseira de aço e mostrador preto.", 825000, "Relógios e Joias", "Vale do Sol Distribuidora"},
	{95, "Relógio Social Mostrador Preto", "Relógio clássico com mostrador preto, data e pulseira de couro.", 4950000, "Relógios e Joias", "Vale do Sol Distribuidora"},
	{96, "Relógio com Fase da Lua", "Relógio com indicação das fases da lua, mostrador branco e caixa rosé.", 7150000, "Relógios e Joias", "Vale do Sol Distribuidora"},
	{97, "Relógio Bicolor com Data", "Relógio de pulseira bicolor com janela de data, de desenho atemporal.", 6050000, "Relógios e Joias", "Vale do Sol Distribuidora"},
	{98, "Relógio de Mergulho Bicolor", "Relógio de mergulho resistente à água, com luneta giratória.", 7700000, "Relógios e Joias", "Vale do Sol Distribuidora"},
	{182, "Brinco de Cristal Verde", "Brinco pendente com cristal verde, para ocasiões especiais.", 16500, "Relógios e Joias", "Vale do Sol Distribuidora"},
	{183, "Brinco Oval Verde", "Brinco de argola oval estampada em verde, moderno e versátil.", 13700, "Relógios e Joias", "Vale do Sol Distribuidora"},
	{184, "Brinco Tropical de Folha", "Brinco em forma de folha, colorido e divertido, com clima de verão.", 11000, "Relógios e Joias", "Vale do Sol Distribuidora"},
	{190, "Relógio Automático de Aço", "Relógio de caixa em aço inoxidável e movimento automático, preciso e sofisticado.", 2750000, "Relógios e Joias", "Vale do Sol Distribuidora"},
	{191, "Relógio Feminino com Fase da Lua", "Relógio com fases da lua, caixa rosé e pulseira de couro.", 8800000, "Relógios e Joias", "Vale do Sol Distribuidora"},
	{192, "Relógio Feminino Bicolor com Data", "Relógio feminino de pulseira bicolor e janela de data, elegante e funcional.", 6050000, "Relógios e Joias", "Vale do Sol Distribuidora"},
	{193, "Relógio Feminino Dourado", "Relógio de caixa dourada e mostrador azul, com um toque de glamour.", 440000, "Relógios e Joias", "Vale do Sol Distribuidora"},
	{194, "Relógio Feminino de Aço", "Relógio de pulseira metálica e desenho simples, para o dia a dia.", 71500, "Relógios e Joias", "Vale do Sol Distribuidora"},

	// Beleza e Perfumaria (13)
	{1, "Máscara para Cílios Volume", "Máscara de efeito volume e alongamento, de longa duração e fórmula sem crueldade animal.", 5500, "Beleza e Perfumaria", "Sertão Empório"},
	{2, "Paleta de Sombras com Espelho", "Paleta com vários tons de sombra e espelho embutido, prática para retocar a maquiagem fora de casa.", 11000, "Beleza e Perfumaria", "Sertão Empório"},
	{3, "Pó Compacto Translúcido", "Pó finíssimo para fixar a maquiagem e controlar o brilho, com acabamento matte.", 8200, "Beleza e Perfumaria", "Sertão Empório"},
	{4, "Batom Vermelho", "Batom cremoso e pigmentado, de cor intensa e duradoura.", 7100, "Beleza e Perfumaria", "Sertão Empório"},
	{5, "Esmalte Vermelho", "Esmalte vermelho brilhante de secagem rápida, com acabamento de salão em casa.", 4900, "Beleza e Perfumaria", "Sertão Empório"},
	{6, "Perfume Unissex Cítrico", "Fragrância unissex de aroma fresco e limpo, para usar todos os dias.", 27500, "Beleza e Perfumaria", "Sertão Empório"},
	{7, "Perfume Noturno Frasco Preto", "Fragrância elegante e misteriosa, com notas de toranja, rosa e sândalo, para a noite.", 71500, "Beleza e Perfumaria", "Sertão Empório"},
	{8, "Perfume Floral Frasco Ânfora", "Fragrância floral com ylang-ylang, rosa e jasmim, feminina e sofisticada.", 49500, "Beleza e Perfumaria", "Sertão Empório"},
	{9, "Perfume Frutado com Laço", "Fragrância vibrante e alegre, com notas de manga, jasmim e madeiras claras.", 38500, "Beleza e Perfumaria", "Sertão Empório"},
	{10, "Perfume Floral Frasco Rosé", "Fragrância floral e romântica, com notas de tuberosa e jasmim.", 44000, "Beleza e Perfumaria", "Sertão Empório"},
	{118, "Sabonete Líquido para Mãos", "Sabonete líquido de origem vegetal que limpa e hidrata, deixando as mãos macias.", 4900, "Beleza e Perfumaria", "Sertão Empório"},
	{119, "Sabonete Líquido Corporal Karité", "Sabonete corporal com manteiga de karité e espuma cremosa, que hidrata e nutre a pele.", 7100, "Beleza e Perfumaria", "Sertão Empório"},
	{120, "Loção Hidratante Masculina", "Loção para corpo e rosto de absorção rápida, que mantém a pele hidratada por mais tempo.", 5500, "Beleza e Perfumaria", "Sertão Empório"},

	// Veículos (10)
	{113, "Motocicleta Esportiva Prata", "Moto versátil e confiável, confortável para vários estilos de pilotagem.", 2200000, "Veículos", "Atlântico Importados"},
	{114, "Motocicleta Urbana Vermelha", "Moto potente e ágil, de visual marcante e desempenho esportivo.", 4950000, "Veículos", "Atlântico Importados"},
	{115, "Motocicleta de Corrida Vermelha", "Moto de alto desempenho inspirada nas pistas de corrida.", 8250000, "Veículos", "Atlântico Importados"},
	{116, "Scooter Urbana Cinza", "Scooter econômica e prática para o deslocamento na cidade, fácil de pilotar.", 1650000, "Veículos", "Atlântico Importados"},
	{117, "Motocicleta Esportiva Verde", "Moto esportiva e aerodinâmica, feita para velocidade e emoção.", 4125000, "Veículos", "Atlântico Importados"},
	{167, "Sedã de Luxo Vermelho", "Sedã confortável e elegante, com acabamento de luxo e rodar suave.", 15950000, "Veículos", "Atlântico Importados"},
	{168, "Sedã Esportivo Azul", "Sedã potente com tração traseira, que une desempenho e praticidade.", 18150000, "Veículos", "Atlântico Importados"},
	{169, "Utilitário Compacto Dourado", "Utilitário compacto e ágil, com um toque esportivo para a cidade.", 13750000, "Veículos", "Atlântico Importados"},
	{170, "Utilitário Esportivo Preto", "Utilitário espaçoso e versátil, com bom desempenho e pensado para a família.", 20350000, "Veículos", "Atlântico Importados"},
	{171, "Minivan Prata", "Minivan bem equipada, com conforto e praticidade para viagens em família.", 17600000, "Veículos", "Atlântico Importados"},
}

// As duas contas de demonstração do NFR-1. O hash é Argon2id no formato PHC,
// com os parâmetros do AD-9 (m=19456, t=2, p=1, sal de 16 bytes) — literal
// porque a semente é determinística, e o sal é fixo pelo mesmo motivo.
// Reproduzir: argon2.IDKey([]byte(senha), []byte(sal), 2, 19456, 1, 32).
var contas = []struct{ tabela, nome, email, senha, hash string }{
	{
		"identidade.comprador", "Joana Ribeiro", "comprador@azamon.test", "azamon-comprador",
		"$argon2id$v=19$m=19456,t=2,p=1$YXphbW9uLWRlbW8tMDAwMQ$6Q3cmzFmai+W5Dz97kfAqozB+u5MVpUkiwZdGJXJXQ8",
	},
	{
		"identidade.administrador", "Marcos Aleixo", "admin@azamon.test", "azamon-admin",
		"$argon2id$v=19$m=19456,t=2,p=1$YXphbW9uLWRlbW8tMDAwMg$rQXJzEZBGDbaO34QvCTVhxFu87lcvsuu4T6Oa85rw8k",
	},
}

// ---------------------------------------------------------------------------

const (
	dirMedia   = "media"
	arquivoSQL = "db/semente/001_catalogo_semeado.sql"
	baseURL    = "/api/v1/media/" // relativa: quem serve é o Go, no compose ou fora dele
)

func main() {
	if err := gerar(); err != nil {
		fmt.Fprintln(os.Stderr, "gerar:", err)
		os.Exit(1)
	}
}

// conferir roda antes de qualquer escrita: nome repetido daria o mesmo uuid,
// apelido repetido daria duas linhas apontando para a mesma foto, e imagem
// ausente daria a foto quebrada na Vitrine.
func conferir() error {
	categoria := map[string]bool{}
	for _, c := range categorias {
		categoria[c] = true
	}
	porNome, porApelido := map[string]int{}, map[string]int{}
	for _, p := range produtos {
		if !categoria[p.categoria] {
			return fmt.Errorf("produto %q aponta para categoria inexistente %q", p.nome, p.categoria)
		}
		if o, ok := porNome[p.nome]; ok {
			return fmt.Errorf("nome %q repetido nas origens %d e %d", p.nome, o, p.origem)
		}
		porNome[p.nome] = p.origem
		a := apelido(p.nome)
		if o, ok := porApelido[a]; ok {
			return fmt.Errorf("apelido %q repetido nas origens %d e %d", a, o, p.origem)
		}
		porApelido[a] = p.origem
		if _, err := os.Stat(filepath.Join(dirMedia, a+".webp")); err != nil {
			return fmt.Errorf("imagem ausente para a origem %d: %w", p.origem, err)
		}
	}
	return nil
}

func gerar() error {
	if err := conferir(); err != nil {
		return err
	}

	var sql strings.Builder
	sql.WriteString("-- Catálogo Semeado — GERADO por media/gerar.go. Não edite à mão:\n")
	sql.WriteString("-- a lista mora no gerador, e `go run media/gerar.go` reescreve este arquivo.\n")
	sql.WriteString("--\n")
	sql.WriteString("-- Todo uuid é literal. A semente roda depois das migrações, uma vez só,\n")
	sql.WriteString("-- dentro da transação que insere o marcador em public.semente.\n\n")

	sql.WriteString("INSERT INTO catalogo.vendedor (id, nome, ativo) VALUES\n")
	linhas := make([]string, 0, len(produtos))
	for _, v := range vendedores {
		linhas = append(linhas, fmt.Sprintf("  (%s, %s, true)", literal(id("vendedor", v)), literal(v)))
	}
	sql.WriteString(strings.Join(linhas, ",\n") + ";\n\n")

	sql.WriteString("-- categoria_pai_id fica nulo no MVP (addendum §6): a coluna existe para\n")
	sql.WriteString("-- que hierarquia depois não seja migração de dados.\n")
	sql.WriteString("INSERT INTO catalogo.categoria (id, nome, categoria_pai_id) VALUES\n")
	linhas = linhas[:0]
	for _, c := range categorias {
		linhas = append(linhas, fmt.Sprintf("  (%s, %s, NULL)", literal(id("categoria", c)), literal(c)))
	}
	sql.WriteString(strings.Join(linhas, ",\n") + ";\n\n")

	sql.WriteString("INSERT INTO catalogo.produto (id, nome, descricao, preco_centavos, imagem_url, vendedor_id, categoria_id, busca_normalizada) VALUES\n")
	linhas = linhas[:0]
	for _, p := range produtos {
		linhas = append(linhas, fmt.Sprintf("  (%s, %s, %s, %d, %s, %s, %s, %s)",
			literal(id("produto", p.nome)), literal(p.nome), literal(p.descricao), p.centavos,
			literal(baseURL+apelido(p.nome)+".webp"), literal(id("vendedor", p.vendedor)), literal(id("categoria", p.categoria)),
			literal(catalogo.NormalizarBusca(p.nome, p.descricao))))
	}
	sql.WriteString(strings.Join(linhas, ",\n") + ";\n\n")

	sql.WriteString("-- Contas de demonstração (FR-4): duas tabelas separadas, nunca uma coluna\n")
	sql.WriteString("-- de papel. As credenciais estão no README.\n")
	for _, c := range contas {
		sql.WriteString(fmt.Sprintf("INSERT INTO %s (id, nome, email, senha_hash) VALUES\n  (%s, %s, %s, %s);\n",
			c.tabela, literal(id("conta", c.email)), literal(c.nome), literal(c.email), literal(c.hash)))
	}

	if err := os.WriteFile(arquivoSQL, []byte(sql.String()), 0o644); err != nil {
		return err
	}
	fmt.Printf("%d Produtos, %d Categorias e %d Vendedores em %s\n", len(produtos), len(categorias), len(vendedores), arquivoSQL)
	return nil
}

// id deriva um uuid estável do nome (formato v5, SHA-1). Estável é o ponto:
// `docker compose down -v` seguido de `up` devolve os mesmos identificadores.
func id(tipo, nome string) string {
	s := sha1.Sum([]byte("azamon:" + tipo + ":" + nome))
	b := s[:16]
	b[6] = b[6]&0x0f | 0x50 // versão 5
	b[8] = b[8]&0x3f | 0x80 // variante RFC 4122
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// literal cita um texto para o SQL. A aspa simples dobrada é o único escape
// que o Postgres pede em literal padrão, e toda entrada daqui é do repositório.
func literal(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// apelido reduz o nome do Produto a um nome de arquivo estável. A caixa e os
// acentos saem por catalogo.Normalizar, a mesma função que preenche
// busca_normalizada — duas tabelas de acento divergindo é defeito à espera.
func apelido(nome string) string {
	var b strings.Builder
	for _, r := range catalogo.Normalizar(nome) {
		switch {
		case r >= 'a' && r <= 'z', unicode.IsDigit(r):
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
