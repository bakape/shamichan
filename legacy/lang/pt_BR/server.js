/*
 * Mapeamento de opções configuráveis de linguagem para o servidor somente
 */
var lang = {
	catalog_omit: 'Respostas/Imagens',
	show_seconds: 'Clique para mostrar os segundos',
	worksBestWith: 'funciona melhor com',

	// Respostas do imager
	im : {
		bad_client: "ID de client ruim.",
		too_large: 'Arquivo grande demais.',
		req_problem: 'Erro ao solicitar upload.',
		aborted: 'Upload foi abortado.',
		received: '% recebidos...',
		invalid: 'Upload inválido.',
		no_image: 'Sem imagem.',
		bad_spoiler: 'Spoiler ruim.',
		temp_tracking: "Erro temporário de rastreamento: ",
		invalid_format: 'Formato de imagem inválido.',
		verifying: 'Verificando...',
		missing: "Arquivo se foi.",
		video_invalid: "Arquivo de vídeo inválido.",
		ffmpeg_too_old: "ffmpeg do servidor é muito velho.",
		mp3_no_cover: 'MP3 não tem arte de capa.',
		video_unknown: "Erro desconhecido ao ler vídeo.",
		video_format: 'Formato de arquivo não permitido.',
		audio_kinshi: 'Áudio não é permitido.',
		bad: 'Imagem malvada.',
		not_png: 'Não é um arquivo PNG ou APNG.',
		video: 'Vídeo',
		image: 'Imagem',
		bad_dims: 'Dimensões de imagem ruins.',
		too_many_pixels: 'Pixels demais.',
		too_wide_and_tall: ' é muito comprida e muito alta.',
		too_wide: ' é muito comprida.',
		too_tall: ' é muito alta.',
		thumbnailing: 'Criando miniatura...',
		not_image: 'Arquivo não é uma imagem',
		unsupported: "Tipo de arquivo não suportado.",
		dims_fail: "Não foi possível ler as dimensões da imagem.",
		hashing: 'Erro ao criar hash.',
		resizing: "Erro de redimensionamento.",
		pngquant: "Erro ao criar miniatura no pngquant.",
		unknown: 'Erro desconhecido ao processar imagem.',
	},
	// Várias strings de template
	tmpl: {
	name: 'Nome:',
		email: 'Email:',
		options: 'Opções',
		identity: 'Identidade',
		faq: 'FAQ',
		schedule: 'Programação',
		feedback: 'Contato',
		onlineCounter: 'Contador Online'
	},
	/*
	 * Opções do client. Essas opções são renderizadas ao gerar templates,
	 * então só são necessárias pelo servidor
	 * id: [rótulo, tooltip]
	 */
	opts: {
	// Estilos de thumbnail
	small: 'pequena',
		sharp: 'sharp', // there's no good translation to this in Portuguese
		hide: 'esconder',
		// Modos de ajustar as imagens
		none: 'nenhum',
		full: 'tamanho completo',
		width: 'ajustar à largura',
		height: 'ajustar à altura',
		both: 'ajustar à ambas',
		// Nomes para as abas de opções
		tabs: ['Geral', 'Aparência', 'Pesquisa', 'Diversão', 'Atalhos'],
		export: [
			'Exportar',
			'Exportar opções para arquivo'
		],
		import: [
			'Importar',
			'Importar opções de arquivo'
		],
		hidden: [
			'Escondidos: 0',
			'Limpa os posts escondidos'
		],
		lang: [
			'Linguagem',
			'Mudar a linguagem da interface'
		],
		inlinefit: [
			'Expansão',
			'Expandir imagens dentro do post principal e redimensionar de acordo com a configuração'
		],
		thumbs: [
			'Miniaturas',
			'Escolha o tipo de miniatura: '
			+ 'Pequena: 125x125, tamanho de arquivo pequeno; '
			+ 'Sharp: 125x125, mais detalhada; '
			+ 'Esconder: esconde todas as imagens;',
		],
		imageHover: [
			'Expansão de Imagem ao Pairar',
			'Mostra prévias de imagens ao pairar'
		],
		webmHover: [
			'Expansão de WebM ao pairar',
			'Mostra prévias de WebM ao pairar. Requer Expansão de Imagem ao Pairar ativado.'
		],
		autogif: [
			'Miniaturas de GIF animadas',
			'Miniaturas de GIF animadas'
		],
		spoilers: [
			'Imagens em spoiler',
			"Não colocar spoilers em imagens"
		],
		linkify: [
			'Links clicáveis',
			'Converte todas as URLs em texto para links clicáveis no post. ATENÇÃO: Potencial'
			+ ' problema de segurança (XSS). Requer que a página seja atualizada.'
		],
		notification: [
			'Notificações de Desktop',
			'Tenha notificações no desktop ao ser quotado ou quando o syncwatch começar'
		],
		anonymise: [
			'Anonimizar',
			'Mostra todos os postadores como anônimos'
		],
		relativeTime: [
			'Estampas de tempo relativas',
			'Estampas de tempo relativas. Ex.: \'1 hora atrás\''
		],
		nowPlaying: [
			'Tocando agora',
			'Música atual da r/a/dio e outras informações de stream'
			+ ' no banner superior.'
		],
		// Custom localisation functions
		imageSearch: [
			function(site) {
				return lang.common.capitalize(site) + ' Pesquisa de Imagens';
			},
			function(site) {
			return `Mostrar
				/ esconder links de pesquisa do(a) ${lang.common.capitalize(site)}`;
			}
		],
		illyaBGToggle: [
			'Illya Dance',
			'Loli dançando no fundo'
		],
		illyaMuteToggle: [
			'Mutar a Illya',
			'Loli dançando sem som'
		],
		horizontalPosting: [
			'Postagem Horizontal',
			'Nostalgia do 38chan'
		],
		replyright: [
			'[Postar] à direita',
			'Move o botão de Postar para a direita da página'
		],
		theme: [
			'Tema',
			'Selecione o tema CSS'
		],
		userBG: [
			'Fundo personalizado',
			'Ativa o fundo personalizado da página'
		],
		userBGimage: [
			'',
			"Imagem para usar como fundo"
		],
		lastn: [
			'[Ùltimos #]',
			'Número de posts a exibir no link de expansão do tópico "Últimos n"'
		],
		postUnloading: [
			' Descarregamento dinâmico de posts',
			'Torna a thread mais resposiva ao descarregar posts' +
			'do topo da thread, para que a contagem de posts se mantenha no valor do Últimos #.'
			+
			'Somente se aplica a threads com o Últimos # ativado.'
		],
		alwaysLock: [
			'Sempre travar no rodapé',
			'Trava o scroll da página no rodapé mesmo com a aba no fundo.'
		],
		// Shortcut keys
		new : [
			'Novo post',
			'Abre um novo post'
		],
		togglespoiler: [
			'Spoiler na imagem',
			'Ativa spoiler no post aberto'
		],
		textSpoiler: [
			'Spoiler de texto',
			'Insere uma tag de spoiler no texto'
		],
		done: [
			'Terminar post',
			'Fecha o post aberto'
		],
		expandAll: [
			'Expandir Todas Imagens',
			'Expande todas as imagens. Webm, PDF, MP3 e seu próprio post'
			+ ' não são afetados. Novos posts de imagem também são expandidos'
		],
		workMode: [
			'Work mode',
			'Hides images, defaults theme and disables user background'
		],
		workModeTOG: [
			'Work mode',
			'Hides images, defaults theme and disables user background'
		]
	}
};
	lang.common = require('./common');
	module.exports = lang;
