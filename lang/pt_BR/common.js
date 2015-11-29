/*
 * Shared by the server and client
 */

var lang = {
	anon: 'Anônimo',
	search: 'Pesquisa',
	show: 'Exibir',
	hide: 'Esconder',
	report: 'Reportar',
	focus: 'Focar',
	expand: 'Expandir',
	last: 'Últimos',
	see_all: 'Ver todos',
	bottom: 'Rodapé',
	expand_images: 'Expandir Imagens',
	live: 'ao vivo',
	catalog: 'Catálogo',
	return: 'Retornar',
	top: 'Topo',
	reply: 'Postar',
	newThread: 'Novo tópico',
	locked_to_bottom: 'Travado ao rodapé',
	you: '(Tu)',
	done: 'Feito',
	send: 'Enviar',
	locked: 'trancado',
	thread_locked: 'Este tópico está trancado.',
	langApplied: 'Configurações de linguagem foram mudadas. Esta página recarregará agora.',
	googleSong: 'Clique para pesquisar (google) a música',
	quoted: 'Você foi quotado',
	syncwatchStarting: 'Syncwatch começará em 10 segundos',
	finished: 'Terminado.',
	expander: ['Expandir imagens', 'Contrair imagens'],
	uploading: 'Enviando...',
	subject: 'Assunto',
	cancel: 'Cancelar',
	unknownUpload: 'Erro de upload desconhecido',
	unknownResult: 'Resultado desconhecido',
	rescan: 'Rescan',

	reports: {
		post: 'Reportando post',
		reporting: 'Reportando...',
		submitted: 'Denúncia enviada!',
		setup: 'Obtendo reCAPTCHA...',
		loadError: 'Não foi possível carregar o reCATPCHA'
	},

	// Time-related
	week: ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sab'],
	year: ['Jan', 'Fev', 'Mar', 'Abr', 'Mai', 'Jun', 'Jul', 'Ago', 'Set',
		'Out', 'Nov', 'Dez'],
	just_now: 'agora mesmo',
	unit_minute: 'minuto',
	unit_hour: 'hora',
	unit_day: 'dia',
	unit_month: 'mês',
	unit_year: 'ano',

	// Websocket syncronisation status
	sync: {
		notSynced: 'Dessincronizado',
		connecting: 'Conectando',
		syncing: 'Sincronizando',
		synced: 'Sincronizado',
		dropped: 'Caiu',
		reconnecting: 'Reconectando'
	},

	// Moderation language map
	mod: {
		title: ['Título', 'Mostra o título de staff nos posts'],
		clearSelection: ['Limpar', 'Limpa a seleção de posts'],
		spoilerImages: ['Spoiler', 'Adiciona spoiler na imagem dos posts selecionados'],
		deleteImages: ['Del Img', 'Deleta a imagem do post selecionado'],
		deletePosts: ['Del Post', 'Deleta o post selecionado'],
		lockThreads: ['Trancar', 'Tranca/destranca o tópico selecionado'],
		toggleMnemonics: ['Mnemonics', 'Ativa a exibição de mnemonics'],
		sendNotification: [
			'Notificação',
			'Envia uma mensagem de notificação para todos os clientes'
		],
		renderPanel: ['Painel', 'Ativa a exibição do painel de administração'],
		ban: ['Ban', 'Ban o(s) usuário(s) pelos posts selecionados'],
		modLog: ['Reg', 'Mostra o registro de moderação'],
		djPanel: ['DJ', 'DJ tool panel'],
		displayBan: [
			'Exibir',
			'Adiciona uma mensagem \'O USUÁRIO FOI BANIDO POR ESTA POSTAGEM\' publicamente'
		],
		banMessage: 'O USUÁRIO FOI BANIDO POR ESTA POSTAGEM',
		unban: 'Desbanir',
		placeholders: {
			msg: 'Mensagem',
			days: 'd',
			hours: 'h',
			minutes: 'min',
			reason: 'Razão'
		},
		needReason: 'É necessário especificar uma razão',

		// Correspond to websocket calls in common/index.js
		7: 'Spoiler adicionado à imagem',
		8: 'Imagem deletada',
		9: 'Postagem deletada',
		10: 'Tópico trancado',
		11: 'Tópico destrancado',
		12: 'Usuário banido',
		53: 'Usuário desbanido',

		// Formatting function for moderation messages
		formatLog: function (act) {
			var msg = lang.mod[act.kind] + ' por ' + act.ident;
			if (act.reason)
				msg += ' pela razão de ' + act.reason;
			return msg;
		}
	},

	// Format functions
	pluralize: function(n, noun) {
		// For words ending with 'y' and not a vovel before that
		if (n != 1
			&& noun.slice(-1) == 'y'
			&& ['a', 'e', 'i', 'o', 'u'].indexOf(noun.slice(-2, -1)
				.toLowerCase()) < 0) {
			noun = noun.slice(0, -1) + 'ie';
		}
		return n + ' ' + noun + (n == 1 ? '' : 's');
	},
	capitalize: function(word) {
		return word[0].toUpperCase() + word.slice(1);
	},
	// 56 minutos atrás
	ago: function(time, unit, isFuture) {
		var res = lang.pluralize(time, unit);
		if (isFuture)
			res = 'em ' + res;
		else
			res += ' atrás';
		return res;
	},
	// 47 respostas and 21 images omited
	abbrev_msg:  function(omit, img_omit, url) {
		var html = lang.pluralize(omit, 'postagen');
		if (img_omit)
			html += ' e ' + lang.pluralize(img_omit, 'imagen');
		html += ' omitidas';
		if (url) {
			html += ' <span class="act"><a href="' + url + '" class="history">'
				+ lang.see_all + '</a></span>';
		}
		return html;
	}
};

module.exports = lang;
