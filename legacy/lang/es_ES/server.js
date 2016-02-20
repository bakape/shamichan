/*
 * Mapa de opciones de idiomas configurables solamente para servidor
 */
var lang = {
	catalog_omit: 'Respuestas/Imagenes',
	show_seconds: 'Click para mostrar segundos',
	worksBestWith: 'funciona mejor con',

	// Respuestas de imager
	im : {
		bad_client: "ID de cliente mala.",
		too_large: 'Imagen es demasiado grande.',
		req_problem: 'Problema con la solicitud de subida.',
		aborted: 'Subida abortada.',
		received: '% recibido...',
		invalid: 'Subida inválida.',
		no_image: 'No imagen.',
		bad_spoiler: 'Spoiler malo.',
		temp_tracking: "Error de rastreamiento temporal: ",
		invalid_format: 'Formato de imagen inválido.',
		verifying: 'Verificando...',
		missing: "Archivo desaparecido.",
		video_invalid: "Archivo de vídeo invalido.",
		ffmpeg_too_old: "ffmpeg del servidor esta demasiado anticuado.",
		mp3_no_cover: 'MP3 no tiene portada.',
		video_unknown: "Error de lectura de vídeo desconocido.",
		video_format: 'Formato de archivo corrupto.',
		audio_kinshi: 'Audio no es permitido.',
		bad: 'Imagen mala.',
		not_png: 'No es PNG o APNG.',
		video: 'Vídeo',
		image: 'Imagen',
		bad_dims: 'Dimensiones de imagen incorrectas.',
		too_many_pixels: 'Demasiados píxeles.',
		too_wide_and_tall: ' es demasiado ancha y larga.',
		too_wide: ' es demasiado ancha.', // No such thing
		too_tall: ' es demasiado alta.',
		thumbnailing: 'Miniaturizando...',
		not_image: 'Archivo no es una imagen',
		unsupported: "Tipo de archivo no soportado.",
		dims_fail: "No se pudo leer las dimensiones de la imagen.",
		hashing: 'Error de hashing.',
		resizing: "Error al redimensionar.",
		pngquant: "Error de minituarización de Pngquant.",
		unknown: 'Error de procesación de imagen desconocido.'
	},

	//Varias strings de template
	tmpl: {
		name: 'Nombre:',
		email: 'Email:',
		options: 'Opciones',
		identity: 'Identidad',
		faq: 'FAQ',
		schedule: 'Programación',
		feedback: 'Contacto',
		onlineCounter: 'Contador online'
	},

	/*
	 * Opciones de cliente. El panel de opciones es renderizado cuando el template es generado, así que
	 * estos son solo necesitados por el servidor
	 * id: [etiqueta, tooltip]
	 */
	opts: {
		// Estilos de thumbnail
		small: 'Pequeño',
		sharp: 'Agudo',
		hide: 'Esconder',
		// Modo de ajuste de imagen
		none: 'ninguno',
		full: 'Tamaño completo',
		width: 'ajustar a anchura',
		height: 'ajustar a altura',
		both: 'ajustar a ambas',

		// Nombres para las pestañas del panel de opciones
		tabs: ['General', 'Estilo', 'Búsqueda', 'Entretenimiento', 'Atajos'],
		export: [
			'Exportar',
			'Exportar opciones a archivo'
		],
		import: [
			'Importar',
			'Importar opciones desde archivo'
		],
		hidden: [
			'Escondido: 0',
			'Limpiar posts escondidos'
		],
		lang: [
			'Idioma',
			'Cambia a diferentes idiomas'
		],
		inlinefit: [
			'Expansión',
			'Expande imágenes dentro del post padre y redimensiona según opciones.'
		],
		thumbs: [
			'Thumbnails',
			'Establecer tamaño de thumbnail: '
				+ 'Pequeño: 125x125, tamaño de imagen pequeño; '
				+ 'Agudo: 125x125, mas detallado; '
				+ 'Esconder: Esconde todas las imágenes;'
		],
		imageHover: [
			'Expansion de imagen al pasar el ratón',
			'Muestra una previsualización de la imagen al pasar'
		],
		webmHover: [
			'Expansión de WebM al pasar el ratón',
			'Muestra una previsualización del WebM al pasar. Requiere tener Expansion de imagen al pasar el ratón activado.'
		],
		autogif: [
			'Thumbnail de GIF animado',
			'Anima thumbnails de GIF'
		],
		spoilers: [
			'Spoiler de imágenes',
			"No spoilear imágenes"
		],
		linkify: [
			'Links clickeables',
			'Convierte texto en el post en link clickeable. PELIGRO: Potencial'
				+ ' problema de seguridad (XSS). Requiere recarga de pagina.'
		],
		notification: [
			'Notificaciones de escritorio',
			'Recibe notificaciones de escritorio cuando eres citado o un syncwatch esta a punto de comenzar'
		],
		anonymise: [
			'Anonimizar',
			'Muestra todos los posters como anónimo'
		],
		relativeTime: [
			'Estampas de tiempo relativas',
			'Estampas de tempo relativas. Ex.: \'1 hora atrás\''
		],
		nowPlaying: [
			'Reproducción actual Banner',
			'La canción reproduciéndose ahora en r/a/dio y otra información sobre el stream en'
				+ ' el banner de encima.'
		],
		// Funciones de localizacion personalizadas
		imageSearch: [
			function(site) {
				return lang.common.capitalize(site)  + ' Búsqueda de imágenes';
			},
			function(site) {
				return `Mostrar/Esconder ${lang.common.capitalize(site)} links de búsqueda`;
			}
		],
		illyaBGToggle: [
			'Illya Dance',
			'Loli bailando en el fondo.'
		],
		illyaMuteToggle: [
			'Mute Illya',
			'Loli bailando sin sonido'
		],
		horizontalPosting: [
			'Posting horizontal',
			'Nostalgia de 38chan'
		],
		replyright: [
			'[Responder] a la derecha',
			' Mueve el botón Responder a la derecha de la pagina'
		],
		theme: [
			'Tema',
			'Selecciona tema de CSS'
		],
		userBG: [
			'Fondo personalizado',
			'Activa fondo de pagina personalizado'
		],
		userBGimage: [
			'',
			"Imagen para usar como fondo personalizado"
		],
		lastn: [
			'[Últimos #]',
			'"Numero de post a mostrar con el botón "Últimos n"'
		],
		postUnloading: [
			'Descarga de posts dinámica',
			'Mejora la capacidad de respuesta del thread mediante descargando posts desde la parte superior'
				+ ' del thread, para que el numero de posts se mantenga dentro de el valor de Últimos #.'
				+ ' Solamente se aplica a threads que tengan Last # activado',
		],
		alwaysLock: [
			'Siempre bloquear a la parte inferior',
			'Bloquea scrolling a la parte inferior de la pagina incluso cuando la pestaña esta escondida'
		],
		// Teclas de atajo
		new: [
			'Nuevo post',
			'Abre nuevo post'
		],
		togglespoiler: [
			'Spoiler de imagen',
			'Activa spoiler en el post abierto'
		],
		textSpoiler: [
			'Spoiler de texto',
			'Inserta tag de spoiler'
		],
		done: [
			'Cierra post',
			'Cierra el post abierto'
		],
		expandAll: [
			'Expandir todas las imágenes',
			'Expande todas las imagenes. Webm, PDF y MP3 y tu propio post'
				+ ' no son afectados. Nuevos posts también son expandidos.'
		],
		workMode: [
			'Modo de trabajo',
			'Esconde las imágenes, activa el tema por defecto y deshabilita el fondo personalizado.'
		],
		workModeTOG: [
			'Modo de trabajo',
			'Esconde las imágenes, activa el tema por defecto y deshabilita el fondo personalizado.'
		]
	}
};

lang.common = require('./common');

module.exports = lang;
