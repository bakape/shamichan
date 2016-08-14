var lang = {
	// Moderation language map
	mod: {
		title: ["Título", "Mostra o título de staff nos posts"],
		clearSelection: ["Limpar", "Limpa a seleção de posts"],
		spoilerImages: ["Spoiler", "Adiciona spoiler na imagem dos posts selecionados"],
		deleteImages: ["Del Img", "Deleta a imagem do post selecionado"],
		deletePosts: ["Del Post", "Deleta o post selecionado"],
		lockThreads: ["Trancar", "Tranca/destranca o tópico selecionado"],
		toggleMnemonics: ["Mnemonics", "Ativa a exibição de mnemonics"],
		sendNotification: [
			"Notificação",
			"Envia uma mensagem de notificação para todos os clientes"
		],
		renderPanel: ["Painel", "Ativa a exibição do painel de administração"],
		ban: ["Ban", "Ban o(s) usuário(s) pelos posts selecionados"],
		modLog: ["Reg", "Mostra o registro de moderação"],
		djPanel: ["DJ", "DJ tool panel"],
		displayBan: [
			"Exibir",
			"Adiciona uma mensagem \"O USUÁRIO FOI BANIDO POR ESTA POSTAGEM\" publicamente"
		],
		banMessage: "O USUÁRIO FOI BANIDO POR ESTA POSTAGEM",
		unban: "Desbanir",
		placeholders: {
			msg: "Mensagem",
			days: "d",
			hours: "h",
			minutes: "min",
			reason: "Razão"
		},
		needReason: "É necessário especificar uma razão",

		// Correspond to websocket calls in common/index.js
		7: "Spoiler adicionado à imagem",
		8: "Imagem deletada",
		9: "Postagem deletada",
		10: "Tópico trancado",
		11: "Tópico destrancado",
		12: "Usuário banido",
		53: "Usuário desbanido",

		// Formatting function for moderation messages
		formatLog: function (act) {
			var msg = lang.mod[act.kind] + " por " + act.ident;
			if (act.reason)
				msg += " pela razão de " + act.reason;
			return msg;
		}
	},
};
