var lang = {
	// Moderation language map
	mod: {
		title: ["Ünvan", "Yönetim derecesini gönderilen girdilerde göster"],
		clearSelection: ["Temizle", "Seçili girdileri temizle"],
		spoilerImages: ["Spoiler", "Seçili resimlere spoiler uygula"],
		deleteImages: ["Res Sil", "Seçili resimleri sil"],
		deletePosts: ["Sil girdi", "Seçili girdileri sil"],
		lockThreads: ["Lock", "Seçili konuları kilitle/me "],
		toggleMnemonics: ["Bellek", "Bellekte tutma sistemi"],
		sendNotification: [
			"Uyarı",
			"Bütün anonlara uyarı gönder"
		],
		ban: ["Ban", "Anon seçili girdi sebebiyle banla"],
		renderPanel: ["Panel", "Yönetim panelini göster"],
		modLog: ["Log", "Moderasyon loglarını göster"],
		djPanel: ["DJ", "DJ paneli"],
		displayBan: [
			"Göster",
			"Ban mesajı ekle \"Kullanıcı bu girdi sebebiyle banlandı\" "
		],
		unban: "Banı kaldır",
		banMessage: "Kullanıcı bu girdi sebebiyle banlandı",
		placeholders: {
			msg: "Mesaj",
			days: "g",
			hours: "s",
			minutes: "dak",
			reason: "Sebep"
		},
		needReason: "Bir sebep olmalı",

		// Correspond to websocket calls in common/index.js
		7: "Resim gizlendi",
		8: "Resim silindi",
		9: "Girdi silindi",
		10: "Konu kilitlendi",
		11: "Konu kilidi açıldı",
		12: "User banned",
		53: "User unbanned",

		// Formatting function for moderation messages
		formatLog: function (act) {
			var msg = lang.mod[act.kind] + ": " + act.ident;
			if (act.reason)
				msg += ": " + act.reason;
			return msg;
		}
	},

	plurals: {
		cevap: "cevaplar",
		resim: "resimler"
	},

	// 47 replies and 21 images omitted
	abbrev_msg:  function(omit, img_omit, url) {
		var html = lang.pluralize(omit, "cevap");
		if (img_omit)
			html += " ve " + lang.pluralize(img_omit, "resim");
		html += " gizlendi";
		if (url) {
			html += " <span class="act"><a href="" + url + "" class="history">"
				+ lang.see_all + "</a></span>";
		}
		return html;
	}
};
