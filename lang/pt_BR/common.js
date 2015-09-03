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
	you: '(You)',
	done: 'Feito',
	send: 'Enviar',

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
		ban: ['Ban', 'Ban poster(s) for the selected post(s)'],
		modLog: ['Reg', 'Mostra o registro de moderação'],
		displayBan: [
			'Display',
			'Append a public \'USER WAS BANNED FOR THIS POST\' message'
		],
		banMessage: 'USER WAS BANNED FOR THIS POST',
		unban: 'Unban',
		placeholders: {
			msg: 'Mensagem',
			days: 'd',
			hours: 'h',
			minutes: 'min',
			reason: 'Reason'
		},
		needReason: 'Must specify reason',

		// Correspond to websocket calls in common/index.js
		7: 'Spoiler adicionado à imagem',
		8: 'Imagem deletada',
		9: 'Postagem deletada',
		10: 'Tópico trancado',
		11: 'Tópico destrancado',
		12: 'User banned',
		53: 'User unbanned',

		// Formatting function for moderation messages
		formatLog: function (act) {
			var msg = lang.mod[act.kind] + ' por ' + act.ident;
			if (act.reason)
				msg += ' for ' + act.reason;
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
	ago: function(time, unit) {
		return lang.pluralize(time, unit) + ' atrás';
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
