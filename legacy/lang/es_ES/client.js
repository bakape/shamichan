var lang = {
	// Mapa de lenguage de moderación
	mod: {
		title: ["Title", "Mostrar tituloe de staff en mis posts nuevos"],
		clearSelection: ["Limpiar", "Limpiar posts seleccionados"],
		spoilerImages: ["Spoiler", "Esconde posts seleccionados"],
		deleteImages: ["Del Img", "Eliminar images de los posts seleccionados"],
		deletePosts: ["Del Post", "Elimina posts seleccionados"],
		lockThreads: ["Cerrar", "Cierra/Abre los hilos seleccionados"],
		toggleMnemonics: ["Mnemónico", "Habilita la muestra de mnemónicos"],
		sendNotification: [
			"Notificación",
			"Enviar mensaje de notificación a todos los clientes"
		],
		renderPanel: ["Panel", "Habilita la muestra de el panel de administrador"],
		ban: ["Expulsar", "Expulsar poster(s) por el numero seleccionado de post(s)"],
		modLog: ["Log", "Mostrar el log de moderación"],
		djPanel: ["DJ", "DJ tool panel"],
		displayBan: [
			"Exponer",
			"Añadir un mensaje \"USUARIO A SIDO EXPULSADO POR ESTE POST\" publico"
		],
		banMessage: "USUARIO A SIDO EXPULSADO POR ESTE POST",
		unban: "Levantar expulsion",
		placeholders: {
			msg: "Mensaje",
			days: "d",
			hours: "h",
			minutes: "min",
			reason: "Motivo"
		},
		needReason: "Debe especificar razon",

		// Corresponde a llamadas de websocket en common/index.js
		7: "Imagen Spoileada",
		8: "Imagen eliminada",
		9: "Post eliminado",
		10: "Hilo cerrado",
		11: "Hilo abierto",
		12: "Usuario expulsado",
		53: "Expulsion levantada",

		// Formato de función para mensajes de moderación
		formatLog: function (act) {
			var msg = lang.mod[act.kind] + " por " + act.ident;
			if (act.reason)
				msg += " por " + act.reason;
			return msg;
		}
	},

	// 47 respuestas y 21 imágenes omitidas
	abbrev_msg:  function(omit, img_omit, url) {
		var html = lang.pluralize(omit, "respuesta");
		if (img_omit)
			html += " y " + lang.pluralize(img_omit, "imagen");
		html += " omitida";
		if (url) {
			html += " <span class="act"><a href="" + url + "" class="history">"
				+ lang.see_all + "</a></span>";
		}
		return html;
	}
};
