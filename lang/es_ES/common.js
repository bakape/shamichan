/*
 * Compartido por el cliente y servidor
 */

var lang = {
	anon: 'Anónimo',
	search: 'Buscar',
	show: 'Mostrar',
	hide: 'Esconder',
	report: 'Reportar',
	focus: 'Centrar',
	expand: 'Ampliar',
	last: 'Últimos',
	see_all: 'Mostrar todos',
	bottom: 'Abajo',
	expand_images: 'Ampliar imágenes',
	live: 'En vivo',
	catalog: 'Catálogo',
	return: 'Regresar',
	top: 'Arriba',
	reply: 'Respuesta',
	newThread: 'Nuevo Hilo',
	locked_to_bottom: 'Pegado al fondo',
	you: '(Tu)',
	done: 'Hecho',
	send: 'Enviar',
	locked: 'Cerrado',
	thread_locked: 'Este hilo esta cerrado.',
	langApplied: 'Opciones de idioma applicadas. La pagina sera recargada ahora.',
	googleSong: 'Clock para googlear la cancion',
	quoted: 'Has sido citado',
	syncwatchStarting: 'Syncwatch empezando end 10 segundos',
	finished: 'Terminado',
	expander: ['Expandir imagenes', 'Contraer imagenes'],
	uploading: 'Subiendo...',
	subject: 'Sujeto',
	cancel: 'Cancelar',
	unknownUpload: 'Error de subida desconocido',
	unknownResult: 'Resultado desconocido',
	rescan: 'Rescan',

	reports: {
		post: 'Reportando post',
		reporting: 'Reportando...',
		submitted: 'Report enviado!',
		setup: 'Obteniendo reCAPTCHA...',
		loadError: 'No se pudo cargar reCATPCHA'
	},

	// Relacionado con el tiempo
	week: ['Dom', 'Lun', 'Mar', 'Mie', 'Jue', 'Vie', 'Sab'],
	year: ['Ene', 'Feb', 'Mar', 'Abr', 'May', 'Jun', 'Jul', 'Ago', 'Sep',
		'Oct', 'Nov', 'Dic'],
	just_now: 'ahora mismo',
	unit_minute: 'minuto',
	unit_hour: 'hora',
	unit_day: 'día',
	unit_month: 'mes',
	unit_year: 'año',

	// Estado de sincronizacion de websockets
	sync: {
		notSynced: 'No sincronizado',
		connecting: 'Conectando',
		syncing: 'Sincronizando',
		synced: 'Sincronizado',
		dropped: 'Caido',
		reconnecting: 'Reconectando'
	},

	// Mapa de lenguage de moderación
	mod: {
		title: ['Title', 'Mostrar tituloe de staff en mis posts nuevos'],
		clearSelection: ['Limpiar', 'Limpiar posts seleccionados'],
		spoilerImages: ['Spoiler', 'Esconde posts seleccionados'],
		deleteImages: ['Del Img', 'Eliminar images de los posts seleccionados'],
		deletePosts: ['Del Post', 'Elimina posts seleccionados'],
		lockThreads: ['Cerrar', 'Cierra/Abre los hilos seleccionados'],
		toggleMnemonics: ['Mnemónico', 'Habilita la muestra de mnemónicos'],
		sendNotification: [
			'Notificación',
			'Enviar mensaje de notificación a todos los clientes'
		],
		renderPanel: ['Panel', 'Habilita la muestra de el panel de administrador'],
		ban: ['Expulsar', 'Expulsar poster(s) por el numero seleccionado de post(s)'],
		modLog: ['Log', 'Mostrar el log de moderación'],
		djPanel: ['DJ', 'DJ tool panel'],
		displayBan: [
			'Exponer',
			'Añadir un mensaje \'USUARIO A SIDO EXPULSADO POR ESTE POST\' publico'
		],
		banMessage: 'USUARIO A SIDO EXPULSADO POR ESTE POST',
		unban: 'Levantar expulsion',
		placeholders: {
			msg: 'Mensaje',
			days: 'd',
			hours: 'h',
			minutes: 'min',
			reason: 'Motivo'
		},
		needReason: 'Debe especificar razon',

		// Corresponde a llamadas de websocket en common/index.js
		7: 'Imagen Spoileada',
		8: 'Imagen eliminada',
		9: 'Post eliminado',
		10: 'Hilo cerrado',
		11: 'Hilo abierto',
		12: 'Usuario expulsado',
		53: 'Expulsion levantada',

		// Formato de función para mensajes de moderación
		formatLog: function (act) {
			var msg = lang.mod[act.kind] + ' por ' + act.ident;
			if (act.reason)
				msg += ' por ' + act.reason;
			return msg;
		}
	},

	// Funciones de formato
	pluralize: function(n, noun) {
		// Para palabras que acaban 'y' y no hay una vocal antes.
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
	ago: function(time, unit, isFuture) {
		var res = lang.pluralize(time, unit);
		if (isFuture)
			res = 'en ' + res;
		else
			res += ' atrás';
		return res;
	},
	// 47 respuestas y 21 imágenes omitidas
	abbrev_msg:  function(omit, img_omit, url) {
		var html = lang.pluralize(omit, 'respuesta');
		if (img_omit)
			html += ' y ' + lang.pluralize(img_omit, 'imagen');
		html += ' omitida';
		if (url) {
			html += ' <span class="act"><a href="' + url + '" class="history">'
				+ lang.see_all + '</a></span>';
		}
		return html;
	}
};

module.exports = lang;
