/*
 * Shared by the server and client
 */

const lang = {
	anon: 'Anónimo',
	search: 'Buscar',
	show: 'Mostrar',
	expand: 'Ampliar',
	last: 'Últimos',
	see_all: 'Mostrar todos',
	bottom: 'Abajo',
	expand_images: 'Ampliar imágenes',
	live: 'En vivo',
	catalog: 'Catálogo',
	return: 'Regresar',
	top: 'Arriba',
	reply: 'Reply',
	newThread: 'New thread',

	// Time-related
	week: ['Dom', 'Lun', 'Mar', 'Mie', 'Jue', 'Vie', 'Sab'],
	year: ['Ene', 'Feb', 'Mar', 'Abr', 'May', 'Jun', 'Jul', 'Ago', 'Sep',
		'Oct', 'Nov', 'Dic'],
	just_now: 'ahora mismo',
	unit_minute: 'minuto',
	unit_hour: 'hora',
	unit_day: 'día',
	unit_month: 'mes',
	unit_year: 'año',

	// Format functions
	pluralize: function(n, noun) {
		// For words ending with 'y' and not a vovel before that
		if (n != 1
			&& noun.slice(-1) == 'n'
			&& ['a', 'i', 'o', 'u'].indexOf(noun.slice(-2, -1)
				.toLowerCase()) < 0) {
			noun = noun.slice(0, -1) + 'nes';
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
	// 47 respuestas y 21 imágenes omitidas
	abbrev_msg:  function(omit, img_omit) {
		return lang.pluralize(omit, 'respuesta')
			+ (img_omit ? ' y ' + lang.pluralize(img_omit, 'imagen') : '')
			+ ' omitida.';
	},
};

module.exports = lang;