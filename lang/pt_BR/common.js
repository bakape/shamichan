/*
 * Shared by the server and client
 */

const lang = {
	anon: 'Anônimo',
	search: 'Pesquisa',
	show: 'Exibir',
	expand: 'Expandir',
	last: 'Últimos',
	see_all: 'Ver todos',
	bottom: 'Rodapé',
	expand_images: 'Expandir Imagens',
	live: 'ao vivo',
	catalog: 'Catálogo',
	return: 'Retornar',
	top: 'Topo',

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
	abbrev_msg:  function(omit, img_omit) {
		return lang.pluralize(omit, 'postagens')
			+ (img_omit ? ' e ' + lang.pluralize(img_omit, 'imagen') : '')
			+ ' omitidas.';
	},
};

module.exports = lang;
