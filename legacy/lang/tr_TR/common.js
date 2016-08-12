/*
 * Shared by the server and client
 */

var lang = {
	anon: 'Anon',
	search: 'Ara',
	show: 'Göster',
	hide: 'Gizle',
	report: 'İspiyonla',
	focus: 'Odakla',
	expand: 'Genişlet',
	last: 'Son',
	see_all: 'Hepsini göster',
	bottom: 'Alt',
	expand_images: 'Resimleri genişlet',
	live: 'canlı',
	catalog: 'Katalog',
	return: 'Geri Dön',
	top: 'Üst',
	reply: 'Cevapla',
	newThread: 'Yeni konu',
	locked_to_bottom: 'Aşağı gönderildi',
	you: '(Sen)',
	done: 'Tamam',
	send: 'Gönder',
	locked: 'kilitli',
	thread_locked: 'Bu konu kilitlendi.',
	langApplied: 'Dil ayarları uygulandı. Şimdi sayfayı yenileyeceğiz.',
	googleSong: 'Şarkıyı googleda aratmak için tıklayın',
	quoted: 'Biri sizden alıntı yaptı',
	syncwatchStarting: 'Syncwatch 10 saniye içinde başlıyor',
	finished: 'Bitti',
	expander: ['Resimleri genişlet', 'Resimleri daralt'],
	uploading: 'Yükleniyor...',
	subject: 'Konu',
	cancel: 'İptal',
	unknownUpload: 'Dosya yükleme hatası',
	unknownResult: 'Bilinemeyen hata',
	rescan: 'Tara',

	reports: {
		post: 'Gönderi ispiyonlanıyor',
		reporting: 'İspiyonlanıyor...',
		submitted: 'İspiyon gönderildi!',
		setup: 'reCAPTCHA alınıyor...',
		loadError: 'reCATPCHA yüklenemedi'
	},

	// Time-related
	week: ['Paz', 'Pzt', 'Sal', 'Çrş', 'Prş', 'Cu', 'Cts'],
	year: ['Oca', 'Şub', 'Mar', 'Nis', 'May', 'Haz', 'Tem', 'Ağu', 'Eyl',
		'Eki', 'Kas', 'Ara'],
	just_now: 'şimdi',
	unit_minute: 'dakika',
	unit_hour: 'saat',
	unit_day: 'gün',
	unit_month: 'ay',
	unit_year: 'yıl',

	// Websocket syncronisation status
	sync: {
		notSynced: 'Senkronize değil',
		connecting: 'Bağlanıyor',
		syncing: 'Seknronize oluyor',
		synced: 'Senkronize',
		dropped: 'Düştü',
		reconnecting: 'Tekrar bağlanıyor'
	},

	// Moderation language map
	mod: {
		title: ['Ünvan', 'Yönetim derecesini gönderilen girdilerde göster'],
		clearSelection: ['Temizle', 'Seçili girdileri temizle'],
		spoilerImages: ['Spoiler', 'Seçili resimlere spoiler uygula'],
		deleteImages: ['Res Sil', 'Seçili resimleri sil'],
		deletePosts: ['Sil girdi', 'Seçili girdileri sil'],
		lockThreads: ['Lock', 'Seçili konuları kilitle/me '],
		toggleMnemonics: ['Bellek', 'Bellekte tutma sistemi'],
		sendNotification: [
			'Uyarı',
			'Bütün anonlara uyarı gönder'
		],
		ban: ['Ban', 'Anon seçili girdi sebebiyle banla'],
		renderPanel: ['Panel', 'Yönetim panelini göster'],
		modLog: ['Log', 'Moderasyon loglarını göster'],
		djPanel: ['DJ', 'DJ paneli'],
		displayBan: [
			'Göster',
			'Ban mesajı ekle \'Kullanıcı bu girdi sebebiyle banlandı\' '
		],
		unban: 'Banı kaldır',
		banMessage: 'Kullanıcı bu girdi sebebiyle banlandı',
		placeholders: {
			msg: 'Mesaj',
			days: 'g',
			hours: 's',
			minutes: 'dak',
			reason: 'Sebep'
		},
		needReason: 'Bir sebep olmalı',

		// Correspond to websocket calls in common/index.js
		7: 'Resim gizlendi',
		8: 'Resim silindi',
		9: 'Girdi silindi',
		10: 'Konu kilitlendi',
		11: 'Konu kilidi açıldı',
		12: 'User banned',
		53: 'User unbanned',

		// Formatting function for moderation messages
		formatLog: function (act) {
			var msg = lang.mod[act.kind] + ': ' + act.ident;
			if (act.reason)
				msg += ': ' + act.reason;
			return msg;
		}
	},

	plurals: {
		dakika: "dakika",
		saat: "saatler",
		gün: "günler",
		ay: "ay",
		yıl: "yıl",
		cevap: "cevaplar",
		resim: "resimler"
	},

	// Format functions
	pluralize: function(n, noun) {
		// For words ending with 'y' and not a vovel before that
		if (n != 1) {
			noun = lang.plurals[noun]
		}
		return n + ' ' + noun
	},
	capitalize: function(word) {
		return word[0].toUpperCase() + word.slice(1);
	},
	// 56 minutes ago / in 56 minutes
	ago: function(time, unit, isFuture) {
		var res = lang.pluralize(time, unit);
		if (isFuture)
			res = 'içinde ' + res;
		else
			res += ' önce';
		return res;
	},
	// 47 replies and 21 images omitted
	abbrev_msg:  function(omit, img_omit, url) {
		var html = lang.pluralize(omit, 'cevap');
		if (img_omit)
			html += ' ve ' + lang.pluralize(img_omit, 'resim');
		html += ' gizlendi';
		if (url) {
			html += ' <span class="act"><a href="' + url + '" class="history">'
				+ lang.see_all + '</a></span>';
		}
		return html;
	}
};

module.exports = lang;
