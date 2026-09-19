// Pacote mockdata guarda as audiências fictícias usadas quando a transcrição e
// a análise rodam em modo mock. Cada caso tem a transcrição e a análise
// correspondente, para que a demo mostre algo coerente e reconhecível por um
// advogado. Nomes, números e valores são inventados.
package mockdata

import (
	"hash/fnv"
	"strings"

	"github.com/pietromedeiros/meirinho/internal/domain"
)

type Caso struct {
	// Chave é procurada no título da audiência para escolher o caso: é o que
	// deixa o seed casar "Instrução — Silva x Banco" com a transcrição certa.
	Chave       string
	Area        domain.AreaDoDireito
	Transcricao string
	Resumo      string
	Sugestao    string
	Pontos      []string
}

// Escolher devolve o caso de uma audiência nova. Primeiro pela chave no título;
// sem correspondência, um caso da mesma área escolhido pelo título (estável:
// a mesma audiência reprocessada recebe o mesmo caso).
func Escolher(area domain.AreaDoDireito, titulo string) Caso {
	t := strings.ToLower(titulo)
	var daArea []Caso
	for _, c := range Casos {
		if c.Area != area {
			continue
		}
		if strings.Contains(t, strings.ToLower(c.Chave)) {
			return c
		}
		daArea = append(daArea, c)
	}
	if len(daArea) == 0 {
		return Casos[0]
	}
	h := fnv.New32a()
	h.Write([]byte(titulo))
	return daArea[int(h.Sum32()%uint32(len(daArea)))]
}

// PorTranscricao acha o caso de uma transcrição que o mock produziu. Com
// transcrição real (Whisper) não há correspondência e volta o primeiro caso
// da área — é o limite de combinar motor real com analyzer mock.
func PorTranscricao(transcricao string, area domain.AreaDoDireito) Caso {
	for _, c := range Casos {
		if c.Transcricao == transcricao {
			return c
		}
	}
	for _, c := range Casos {
		if c.Area == area {
			return c
		}
	}
	return Casos[0]
}

var Casos = []Caso{
	{
		Chave: "Silva",
		Area:  domain.AreaCivel,
		Transcricao: `Juiz: Declaro aberta a audiência de instrução e julgamento. Presentes o autor, Carlos Eduardo Silva, e a preposta do réu, Banco Horizonte S.A., ambos acompanhados de seus procuradores.

Juiz: Tentada a conciliação, as partes não chegaram a acordo. Passo à instrução.

Advogado do autor: Excelência, reitero os termos da inicial. Os extratos juntados às fls. 45 a 62 demonstram três descontos não autorizados na conta do autor, somando R$ 18.400,00, entre março e maio.

Advogada do réu: Impugno os extratos. Sustento que as operações foram realizadas com cartão e senha pessoal, o que afasta a falha na prestação do serviço.

Juiz: A preposta confirma que o banco não possui registro de geolocalização das transações?

Preposta: Não tenho essa informação, Excelência.

Juiz: Consigno que a preposta desconhece os registros técnicos das operações. Ouço a testemunha do autor.

Testemunha: Eu estava com o Carlos no dia 14 de março, no hospital, das oito da manhã até a noite. Ele não saiu de lá.

Juiz: Encerrada a instrução. Concedo às partes o prazo sucessivo de 15 dias para memoriais, iniciando pelo autor. Intimem-se.`,
		Resumo: "Audiência de instrução em ação indenizatória por descontos não autorizados " +
			"(R$ 18.400,00) contra o Banco Horizonte. Sem acordo. A preposta do banco não soube " +
			"informar os registros técnicos das transações, o que ficou consignado em ata. A " +
			"testemunha do autor o situou no hospital durante todo o dia 14/03, data de um dos " +
			"descontos. Instrução encerrada, com prazo sucessivo de 15 dias para memoriais, " +
			"começando pelo autor.",
		Sugestao: "Nos memoriais, centrar a tese na inversão do ônus da prova (art. 6º, VIII, " +
			"do CDC) e na responsabilidade objetiva do banco por fortuito interno (Súmula 479 do STJ). " +
			"O desconhecimento da preposta sobre geolocalização e logs deve ser explorado como " +
			"confissão de que o banco não se desincumbiu do ônus. Reforçar o álibi do dia 14/03 com " +
			"o prontuário hospitalar, se possível juntá-lo.",
		Pontos: []string{
			"Prazo de 15 dias para memoriais corre primeiro para o autor.",
			"A tese do banco (uso de cartão e senha) só foi rebatida para um dos três descontos.",
			"Pedir o prontuário do hospital para corroborar o depoimento da testemunha.",
			"Avaliar pedido de dano moral in re ipsa pelo desconto indevido em conta salário.",
		},
	},
	{
		Chave: "alimentos",
		Area:  domain.AreaFamilia,
		Transcricao: `Juíza: Aberta a audiência de conciliação na ação de alimentos proposta por M. S. O., representada pela genitora, em face de Ricardo Oliveira.

Advogada da autora: Excelência, a menor tem hoje 7 anos, frequenta escola particular e faz acompanhamento fonoaudiológico. As despesas comprovadas somam R$ 3.200,00 mensais. Pedimos a fixação em 30% dos rendimentos líquidos do réu.

Advogado do réu: O réu é autônomo, tem renda variável em torno de R$ 6.000,00 e constituiu nova família, com outro filho de 1 ano. Oferece R$ 1.100,00.

Juíza: O senhor declarou imposto de renda no último exercício?

Réu: Declarei, Excelência. Uns noventa mil no ano.

Juíza: Isso dá mais de R$ 7.000,00 por mês, acima do que foi informado. Consigno a divergência.

Advogada da autora: Requeremos a expedição de ofício à Receita Federal e a quebra do sigilo bancário dos últimos 12 meses.

Juíza: Sem acordo. Mantenho os alimentos provisórios em 25% dos rendimentos. Defiro o ofício à Receita. Réu intimado para contestar no prazo de 15 dias.`,
		Resumo: "Audiência de conciliação em ação de alimentos para menor de 7 anos. A autora " +
			"pede 30% dos rendimentos líquidos, com despesas comprovadas de R$ 3.200,00 mensais; o " +
			"réu ofereceu R$ 1.100,00, alegando renda de R$ 6.000,00 e novo filho. O próprio réu " +
			"admitiu ter declarado cerca de R$ 90 mil no último IR, o que indica renda superior à " +
			"informada. Provisórios mantidos em 25%; deferido ofício à Receita Federal.",
		Sugestao: "Explorar a contradição entre a renda alegada e a declarada ao Fisco: ela " +
			"fragiliza a credibilidade do réu em todo o binômio necessidade-possibilidade. Reiterar a " +
			"quebra do sigilo bancário, ainda não apreciada, e pedir a juntada de extratos de " +
			"maquininha de cartão, comuns em renda de autônomo. O novo filho reduz a possibilidade, " +
			"mas não justifica fixação abaixo dos 25% provisórios.",
		Pontos: []string{
			"Réu tem 15 dias para contestar a partir da audiência.",
			"Pedido de quebra de sigilo bancário ficou pendente de decisão.",
			"Organizar os comprovantes de escola e fonoaudiologia para a fase de instrução.",
		},
	},
	{
		Chave: "Anderson",
		Area:  domain.AreaCriminal,
		Transcricao: `Juiz: Aberta a audiência de instrução no processo em que é réu Anderson Lima, denunciado por roubo majorado.

Promotora: Ouço a vítima. A senhora reconhece o réu como autor do fato?

Vítima: Estava escuro, ele usava boné. Na delegacia me mostraram uma foto só, e eu disse que parecia ele.

Defensor: Consigne-se, Excelência, que o reconhecimento na fase policial foi feito por fotografia única, sem as cautelas do artigo 226 do Código de Processo Penal.

Juiz: Consignado.

Policial militar: Abordamos o réu cerca de quarenta minutos depois, a dois quilômetros do local. Não foi encontrado o celular da vítima nem arma com ele.

Defensor: O senhor viu o réu no local dos fatos?

Policial: Não, chegamos a ele pela descrição passada via rádio.

Juiz: Réu, deseja ser interrogado?

Réu: Eu estava no trabalho até as dez da noite. Meu patrão pode confirmar.

Juiz: Defiro a oitiva do empregador como testemunha do juízo. Redesigno a continuação. Mantida a prisão preventiva por ora; a defesa poderá renovar o pedido após a oitiva.`,
		Resumo: "Audiência de instrução em processo por roubo majorado. A vítima admitiu que o " +
			"reconhecimento na delegacia foi feito com uma única fotografia e que não viu bem o " +
			"autor. O policial confirmou que nada foi apreendido com o réu e que a abordagem se " +
			"baseou apenas na descrição passada via rádio. O réu alegou álibi de trabalho; o juiz " +
			"deferiu a oitiva do empregador e manteve a preventiva por ora.",
		Sugestao: "A nulidade do reconhecimento fotográfico sem as cautelas do art. 226 do CPP " +
			"é o eixo da defesa (entendimento consolidado no STJ, HC 598.886/SC). Sem o " +
			"reconhecimento, não resta prova autônoma de autoria: nada foi apreendido. Renovar o " +
			"pedido de revogação da preventiva logo após a oitiva do empregador e preparar o pedido " +
			"de absolvição por insuficiência de provas (art. 386, VII, do CPP).",
		Pontos: []string{
			"Réu segue preso: renovar o pedido de liberdade assim que o álibi for confirmado.",
			"Obter com o empregador o registro de ponto ou as imagens do dia dos fatos.",
			"Fotografia única na delegacia está consignada em ata; citar isso nas alegações.",
			"Nenhum bem da vítima nem arma foi apreendido com o réu.",
		},
	},
	{
		Chave: "Juliana",
		Area:  domain.AreaTrabalhista,
		Transcricao: `Juíza: Aberta a audiência una. Reclamante Juliana Pereira; reclamada Logística Rápida Ltda., representada por preposto.

Juíza: Proposta de conciliação?

Advogado da reclamada: A empresa oferece R$ 12.000,00.

Advogada da reclamante: Recusamos. O pedido de horas extras supera R$ 40.000,00.

Juíza: Sem acordo. Depoimento da reclamante.

Reclamante: Eu entrava às sete e saía em média às oito da noite, com meia hora de almoço. O ponto era batido pelo supervisor, não por nós.

Juíza: Preposto, quem registrava o ponto?

Preposto: Cada funcionário registrava o seu, no sistema.

Testemunha da reclamante: Eu trabalhei no mesmo setor. O supervisor fechava o ponto às cinco e a gente continuava carregando caminhão.

Juíza: A reclamada juntou os cartões de ponto?

Advogado da reclamada: Juntaremos no prazo, Excelência.

Juíza: A empresa tem mais de 20 empregados; os controles deveriam estar nos autos com a defesa. Concedo 5 dias para a juntada, sob as penas da Súmula 338 do TST. Razões finais em 10 dias.`,
		Resumo: "Audiência una em reclamação por horas extras. A reclamada ofereceu R$ 12.000,00, " +
			"recusado. A reclamante relatou jornada das 7h às 20h com 30 minutos de intervalo e " +
			"ponto registrado pelo supervisor; a testemunha confirmou que o ponto era fechado às 17h " +
			"com trabalho posterior. A reclamada não juntou os cartões de ponto com a defesa; a juíza " +
			"concedeu 5 dias sob as penas da Súmula 338 do TST.",
		Sugestao: "Se os cartões não vierem em 5 dias, requerer a presunção de veracidade da " +
			"jornada da inicial (Súmula 338, I, do TST). Se vierem, impugnar os horários uniformes " +
			"(\"britânicos\"), que também invertem o ônus (Súmula 338, III). Incluir nas razões finais " +
			"o intervalo intrajornada suprimido (art. 71, § 4º, da CLT) e os reflexos. A proposta de " +
			"R$ 12 mil está muito abaixo do valor provável da condenação.",
		Pontos: []string{
			"Acompanhar a juntada dos cartões de ponto em 5 dias.",
			"Razões finais em 10 dias.",
			"Depoimento da testemunha confirma o ponto fechado pelo supervisor.",
			"Calcular o intervalo intrajornada suprimido (30 min/dia).",
		},
	},
	{
		Chave: "Alvorada",
		Area:  domain.AreaAdministrativo,
		Transcricao: `Juiz: Audiência na ação anulatória movida por Construtora Alvorada Ltda. contra o Município de Santa Rita, questionando a sanção de impedimento de licitar por dois anos.

Advogado da autora: Excelência, a empresa foi punida sem que lhe fosse dada oportunidade de defesa prévia. A notificação foi enviada a endereço antigo, apesar de o cadastro atualizado constar no próprio portal do município.

Procuradora do município: A notificação seguiu o endereço do contrato. Cabia à empresa manter os dados atualizados.

Juiz: Existe cláusula contratual indicando o portal como meio oficial de comunicação?

Procuradora: Há previsão de comunicação eletrônica, mas não exclusiva.

Testemunha, servidor da comissão: O prazo de defesa correu sem manifestação. A penalidade foi aplicada pelo secretário no dia seguinte, com base no relatório.

Advogado da autora: O relatório menciona que a comissão sequer tentou contato por e-mail?

Testemunha: Não houve tentativa por e-mail.

Juiz: Consignado. Mantenho a tutela que suspendeu os efeitos da sanção. Prazo comum de 15 dias para alegações finais.`,
		Resumo: "Audiência em ação anulatória de sanção de impedimento de licitar (2 anos) aplicada " +
			"pelo Município de Santa Rita. A empresa alega cerceamento de defesa: a notificação foi " +
			"para endereço desatualizado, embora o cadastro correto estivesse no portal do município. " +
			"O servidor da comissão confirmou que não houve tentativa de contato por e-mail e que a " +
			"pena foi aplicada no dia seguinte ao fim do prazo. A tutela suspensiva foi mantida.",
		Sugestao: "A falta de tentativa de comunicação eletrônica, estando o dado disponível " +
			"para a própria Administração, reforça a violação ao contraditório e à ampla defesa (art. " +
			"5º, LV, da CF, e art. 158 da Lei 14.133/2021). Nas alegações finais, destacar a " +
			"desproporcionalidade da sanção e pedir, subsidiariamente, sua redução. A manutenção da " +
			"tutela sinaliza receptividade do juízo à tese.",
		Pontos: []string{
			"Prazo comum de 15 dias para alegações finais.",
			"Juntar print do cadastro atualizado no portal com data anterior à notificação.",
			"Tutela suspensiva mantida: a empresa pode seguir licitando por ora.",
		},
	},
	{
		Chave: "Condomínio",
		Area:  domain.AreaCivel,
		Transcricao: `Juíza: Aberta a audiência de instrução na ação do Condomínio Jardim das Flores contra a Construtora Prisma, por vícios construtivos.

Advogada do condomínio: Excelência, o laudo pericial confirma infiltrações em 14 das 32 unidades e fissuras na fachada, com custo de reparo estimado em R$ 486.000,00.

Advogado da construtora: A construtora impugna o laudo. As infiltrações decorrem da falta de manutenção das juntas de dilatação, obrigação do condomínio prevista no manual do proprietário.

Juíza: Senhor perito, é possível distinguir falha de execução de falta de manutenção?

Perito: Nas fachadas norte e leste, a argamassa foi aplicada com espessura abaixo da norma técnica. Isso é execução. Nas juntas da cobertura, há sinais de falta de manutenção, mas elas respondem por uma parte menor dos danos.

Advogada do condomínio: O prédio foi entregue há três anos, dentro da garantia de cinco anos do artigo 618 do Código Civil.

Juíza: Consignado. Defiro prazo de 10 dias para os assistentes técnicos se manifestarem sobre os esclarecimentos. Depois, alegações finais em 15 dias.`,
		Resumo: "Audiência de instrução em ação por vícios construtivos (infiltrações em 14 de 32 unidades e " +
			"fissuras na fachada; reparo estimado em R$ 486 mil). A construtora atribui os danos à falta de " +
			"manutenção. O perito separou as causas: nas fachadas norte e leste houve falha de execução " +
			"(argamassa abaixo da norma técnica); nas juntas da cobertura, falta de manutenção, com peso menor. " +
			"Prazo de 10 dias para os assistentes técnicos e, depois, 15 dias para alegações finais.",
		Sugestao: "O esclarecimento do perito é favorável: ele atribui a maior parte dos danos a falha de " +
			"execução. Nas alegações finais, ancorar o pedido na garantia de cinco anos do art. 618 do CC e no " +
			"CDC, se houver unidades com adquirentes consumidores. Antecipar a tese de culpa concorrente pedindo " +
			"que eventual redução fique limitada ao custo das juntas da cobertura, que o perito quantificou à parte.",
		Pontos: []string{
			"Assistente técnico tem 10 dias para se manifestar sobre os esclarecimentos.",
			"Separar no orçamento o custo das fachadas e o das juntas da cobertura.",
			"Juntar as atas de assembleia que comprovem a manutenção feita.",
		},
	},
	{
		Chave: "plano de saúde",
		Area:  domain.AreaCivel,
		Transcricao: `Juiz: Audiência de conciliação na ação de Beatriz Nogueira contra a Vida Plena Saúde, sobre negativa de cobertura de cirurgia bariátrica.

Advogado da autora: A autora tem IMC 42, com diabetes tipo 2 e hipertensão. Há relatório médico indicando a cirurgia. A negativa foi por "ausência de cumprimento de protocolo", sem dizer qual.

Preposta da operadora: O protocolo exige dois anos de acompanhamento clínico comprovado. A autora apresentou um ano e quatro meses.

Juiz: Esse requisito consta do contrato ou da diretriz da ANS?

Preposta: Da diretriz interna da operadora.

Advogado da autora: Então é exigência que não está no rol nem no contrato. A tutela de urgência já foi deferida e a operadora não cumpriu em 5 dias.

Juiz: A operadora oferece acordo?

Preposta: Podemos autorizar a cirurgia, sem dano moral.

Advogado da autora: A autora aceita a autorização em 10 dias, mas mantém o pedido de dano moral e a multa pelo descumprimento da tutela.

Juiz: Acordo parcial homologado quanto à obrigação de fazer. O processo segue quanto ao dano moral e às astreintes. Réplica em 15 dias.`,
		Resumo: "Conciliação em ação contra operadora de saúde por negativa de cirurgia bariátrica (IMC 42, com " +
			"comorbidades). A preposta admitiu que a exigência de dois anos de acompanhamento vem de diretriz " +
			"interna, sem previsão no contrato nem na ANS. Houve acordo parcial: autorização da cirurgia em 10 " +
			"dias, homologada. Seguem em discussão o dano moral e a multa pelo descumprimento da tutela.",
		Sugestao: "A confissão de que o requisito é interno e não está no contrato fortalece o dano moral: " +
			"negativa abusiva de tratamento indicado por médico assistente é dano presumido na jurisprudência do " +
			"STJ. Na réplica, liquidar as astreintes desde o fim do prazo de 5 dias da tutela, com certidão de " +
			"descumprimento. Acompanhar o cumprimento dos 10 dias do acordo: se atrasar, peticionar de imediato.",
		Pontos: []string{
			"Operadora tem 10 dias para autorizar a cirurgia (acordo homologado).",
			"Réplica em 15 dias sobre dano moral e astreintes.",
			"Pedir certidão do descumprimento da tutela para calcular a multa.",
		},
	},
	{
		Chave: "Helena",
		Area:  domain.AreaFamilia,
		Transcricao: `Juíza: Audiência no inventário de Helena Castro. Presentes a inventariante, Clara Castro, e o herdeiro Paulo Castro, com procuradores.

Advogado de Paulo: Excelência, o apartamento da Rua das Acácias foi avaliado em R$ 620.000,00, mas há proposta de compra por R$ 780.000,00. A avaliação está defasada.

Advogada da inventariante: A proposta não é firme; é uma carta de intenção sem sinal. Além disso, Paulo morou no imóvel por quatro anos sem pagar aluguel aos demais herdeiros.

Advogado de Paulo: Meu cliente pagou IPTU e condomínio nesse período, cerca de R$ 58.000,00.

Juíza: Há acordo quanto à partilha dos outros bens?

Advogada da inventariante: Sim, das aplicações financeiras e do veículo, em partes iguais.

Juíza: Homologo a partilha parcial das aplicações e do veículo. Quanto ao imóvel, determino nova avaliação por perito do juízo. Os herdeiros terão 15 dias para apresentar os comprovantes das despesas e o pedido de compensação pelo uso exclusivo.`,
		Resumo: "Audiência em inventário com divergência sobre o valor do principal imóvel (avaliação de R$ 620 " +
			"mil contra carta de intenção de R$ 780 mil) e sobre o uso exclusivo do bem por um dos herdeiros " +
			"durante quatro anos. Homologada a partilha parcial das aplicações e do veículo. A juíza determinou " +
			"nova avaliação do imóvel e deu 15 dias para comprovantes de despesas e pedido de compensação.",
		Sugestao: "Representando a inventariante, formular o pedido de arbitramento de aluguel pelo uso exclusivo " +
			"(art. 1.319 do CC), compensando apenas o IPTU e o condomínio efetivamente comprovados; " +
			"condomínio costuma ser encargo de quem usa o imóvel. Indicar assistente técnico para a nova " +
			"avaliação e juntar anúncios de imóveis comparáveis no mesmo bairro.",
		Pontos: []string{
			"15 dias para comprovantes de despesas e pedido de compensação.",
			"Indicar assistente técnico para a nova avaliação.",
			"Partilha das aplicações e do veículo já homologada.",
		},
	},
	{
		Chave: "guarda",
		Area:  domain.AreaFamilia,
		Transcricao: `Juíza: Audiência de instrução na ação de modificação de guarda proposta por Fernando Duarte contra Luciana Ramos, referente ao filho Pedro, de 9 anos.

Psicóloga do juízo: O estudo psicossocial indica vínculo forte da criança com os dois genitores. Pedro relatou desejo de passar mais tempo com o pai, sobretudo nos fins de semana, e ansiedade com as mudanças frequentes de escola.

Advogada da ré: A mãe mudou de cidade por oferta de emprego. A escola nova é melhor avaliada.

Advogado do autor: Foram três mudanças de escola em dois anos. O pai mora a quinze minutos da escola anterior, onde Pedro tem amigos e acompanhamento pedagógico.

Juíza: Os senhores considerariam guarda compartilhada com residência fixa definida pelo critério escolar?

Advogada da ré: A mãe aceita discutir, desde que as férias sejam divididas.

Juíza: Concedo 15 dias para as partes apresentarem proposta conjunta de plano de convivência. Não havendo acordo, o processo vai concluso para sentença.`,
		Resumo: "Instrução em ação de modificação de guarda de criança de 9 anos. O estudo psicossocial aponta " +
			"vínculo forte com ambos os genitores, vontade da criança de ficar mais tempo com o pai e ansiedade " +
			"com as mudanças de escola (três em dois anos). A juíza sinalizou guarda compartilhada com residência " +
			"definida pelo critério escolar e deu 15 dias para as partes apresentarem um plano de convivência.",
		Sugestao: "O juízo sinalizou claramente a solução. Apresentar proposta de plano de convivência detalhado, " +
			"com residência junto ao pai (estabilidade escolar, que o laudo destaca), convivência ampla da mãe, " +
			"férias divididas e calendário de datas especiais. Proposta concreta e equilibrada tende a ser " +
			"adotada na sentença mesmo sem a concordância da outra parte.",
		Pontos: []string{
			"15 dias para a proposta conjunta de plano de convivência.",
			"Laudo psicossocial favorece a estabilidade escolar.",
			"Mãe condicionou o acordo à divisão das férias.",
		},
	},
	{
		Chave: "embriaguez",
		Area:  domain.AreaCriminal,
		Transcricao: `Juiz: Audiência de instrução no processo contra Rodrigo Almeida, denunciado por embriaguez ao volante, artigo 306 do Código de Trânsito.

Promotor: Policial, como foi a abordagem?

Policial rodoviário: Blitz de rotina, às duas da manhã. O condutor apresentava olhos avermelhados e odor etílico. Recusou o etilômetro. Lavramos o auto de constatação de sinais de alteração da capacidade psicomotora.

Defensora: O senhor registrou no auto alteração na fala ou no equilíbrio?

Policial: Marquei odor etílico e olhos vermelhos. Fala e equilíbrio estavam normais.

Defensora: O condutor se envolveu em acidente ou dirigia de forma irregular?

Policial: Não. Foi parado na blitz.

Réu: Eu tinha tomado uma cerveja no jantar, umas quatro horas antes. Recusei o bafômetro porque não sabia que podia fazer depois.

Juiz: Encerrada a instrução. Alegações finais por memoriais em 5 dias, primeiro a acusação, depois a defesa.`,
		Resumo: "Instrução em processo por embriaguez ao volante (art. 306 do CTB). O réu recusou o etilômetro e " +
			"a prova é o auto de constatação. O policial admitiu que registrou apenas odor etílico e olhos " +
			"avermelhados, com fala e equilíbrio normais, e que não houve direção anormal nem acidente. " +
			"Instrução encerrada; memoriais em 5 dias, primeiro a acusação.",
		Sugestao: "Sustentar a absolvição por ausência de prova da alteração da capacidade psicomotora: odor " +
			"etílico e olhos avermelhados, isolados, não bastam segundo a Resolução 432/2013 do Contran, que exige " +
			"conjunto de sinais. O próprio policial afirmou que fala e equilíbrio estavam normais. " +
			"Subsidiariamente, pedir a pena mínima e a substituição por restritiva de direitos.",
		Pontos: []string{
			"Memoriais da defesa em 5 dias, após os da acusação.",
			"Policial confirmou fala e equilíbrio normais: citar trecho da ata.",
			"Não houve acidente nem direção anormal.",
		},
	},
	{
		Chave: "Marcos",
		Area:  domain.AreaTrabalhista,
		Transcricao: `Juiz: Audiência de instrução. Reclamante Marcos Souza; reclamada Supermercados Bom Preço.

Juiz: A perícia concluiu que o trabalho na câmara fria gerava insalubridade em grau médio, sem EPI adequado nos dois primeiros anos. A reclamada tem algo a acrescentar?

Advogada da reclamada: A empresa fornecia japona térmica desde a admissão. Juntamos as fichas de EPI.

Advogado do reclamante: As fichas só têm assinatura a partir de março de 2023. O reclamante foi admitido em 2021.

Juiz: Depoimento do reclamante.

Reclamante: Eu entrava na câmara umas vinte vezes por dia, para repor o estoque de congelados. A japona só chegou depois que um colega ficou doente.

Testemunha da reclamada: Eu sou encarregado. Sempre teve japona, mas nem todo mundo usava.

Advogado do reclamante: O senhor fiscalizava o uso?

Testemunha: Não tinha fiscalização formal.

Juiz: Encerrada a instrução. Razões finais em 10 dias.`,
		Resumo: "Instrução em reclamação por adicional de insalubridade (câmara fria). A perícia concluiu por " +
			"insalubridade em grau médio sem EPI adequado nos dois primeiros anos. As fichas de EPI da reclamada " +
			"só têm assinatura a partir de março de 2023, dois anos após a admissão. O encarregado ouvido como " +
			"testemunha admitiu que não havia fiscalização do uso. Razões finais em 10 dias.",
		Sugestao: "A prova converge: laudo favorável, fichas de EPI com lacuna de dois anos e a testemunha da " +
			"própria reclamada admitindo a falta de fiscalização (a Súmula 289 do TST exige que o empregador " +
			"fiscalize o uso). Nas razões finais, pedir o adicional de 20% no período anterior a março de 2023, " +
			"com reflexos, e questionar a eficácia do EPI depois disso.",
		Pontos: []string{
			"Razões finais em 10 dias.",
			"Fichas de EPI só a partir de 03/2023.",
			"Testemunha da reclamada admitiu que não havia fiscalização do uso do EPI.",
		},
	},
	{
		Chave: "PAD",
		Area:  domain.AreaAdministrativo,
		Transcricao: `Juíza: Audiência na ação ordinária de Sérgio Mendes, servidor estadual demitido após processo administrativo disciplinar, contra o Estado.

Advogado do autor: Excelência, a comissão do PAD foi composta por dois servidores em estágio probatório. A lei estadual exige servidores estáveis.

Procurador do Estado: A irregularidade é formal e não trouxe prejuízo à defesa. O autor foi ouvido e apresentou defesa escrita.

Testemunha, membro da comissão: Eu tinha oito meses de serviço quando fui designado. A presidente da comissão era estável.

Advogado do autor: A comissão ouviu as testemunhas de defesa?

Testemunha: Ouvimos duas das quatro. As outras não foram localizadas no endereço informado.

Advogado do autor: Houve tentativa de intimação por telefone ou e-mail?

Testemunha: Não.

Juíza: Consignado. Prazo de 15 dias para alegações finais.`,
		Resumo: "Audiência em ação anulatória de demissão de servidor após PAD. Um membro da comissão confirmou " +
			"que estava em estágio probatório, contrariando a exigência legal de servidores estáveis. Também " +
			"admitiu que apenas duas das quatro testemunhas de defesa foram ouvidas, sem tentativa de intimação " +
			"por outros meios. Prazo de 15 dias para alegações finais.",
		Sugestao: "Duas nulidades autônomas. A composição irregular da comissão viola a regra legal de " +
			"estabilidade e é vício de competência, não mera formalidade. A não oitiva de metade das testemunhas " +
			"de defesa, sem esforço de localização, mostra prejuízo concreto e afasta o argumento do Estado. " +
			"Pedir a reintegração com efeitos financeiros retroativos à data da demissão.",
		Pontos: []string{
			"Alegações finais em 15 dias.",
			"Obter a portaria de designação da comissão e a data de posse dos membros.",
			"Duas testemunhas de defesa não foram ouvidas: prejuízo concreto.",
		},
	},
}
