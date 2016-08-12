/*
 * Server-only mapping of configurable language options
 */
var lang = {
	catalog_omit: 'Cevaplar/Resimler',
	show_seconds: 'Saniyeleri göstermek için tıklayın',
	worksBestWith: 'en iyi bu browserlar ile uyumludur',

	// Imager responses
	im : {
		bad_client: "Client ID hatalı.",
		too_large: 'Dosya boyutu çok büyük',
		req_problem: 'Yükleme problemi.',
		aborted: 'Yükleme iptal edildi.',
		received: '% alındı...',
		invalid: 'Geçersiz yükleme.',
		no_image: 'Resim yok.',
		bad_spoiler: 'Hatalı spoiler.',
		temp_tracking: "Geçici takip hatası: ",
		invalid_format: 'Geçersiz resim formatı.',
		verifying: 'Doğrulanıyor...',
		missing: "Dosya kayboldu.",
		video_invalid: "Geçersiz video dosyası.",
		ffmpeg_too_old: "Sunucumuzun ffmpeg versiyonu çok eski.",
		mp3_no_cover: 'MP3 kapak resmi yok.',
		video_unknown: "Bilinmeyen video okuma hatası.",
		video_format: 'Dosya formatı hatalı.',
		audio_kinshi: 'Ses yüklemeye izin verilmiyor.',
		bad: 'Hatalı resim.',
		not_png: 'PNG ya da APNG değil.',
		video: 'Video',
		image: 'Resim',
		bad_dims: 'Resim ebatları hatalı.',
		too_many_pixels: 'Çok fazla pixel var gibi',
		too_wide_and_tall: ' çok geniş ve çok uzun.',
		too_wide: ' çok geniş.', // No such thing
		too_tall: ' çok uzun.',
		thumbnailing: 'Küçük resim oluşturuluyor...',
		not_image: 'Dosya bir resim değil',
		unsupported: "Desteklenmeyen dosya biçimi",
		dims_fail: "Resim ebatlarını belirleyemiyoruz.",
		hashing: 'Hash hatası.',
		resizing: "Yeniden boyutlandırma hatası.",
		pngquant: "Pngquant küçük resimleme hatası.",
		unknown: 'Bilinmeyen resim işleme hatası.'
	},

	//Various template strings
	tmpl: {
		name: 'İsim:',
		email: 'E-posta:',
		options: 'Ayarlar',
		identity: 'Kimlik',
		faq: 'SSS',
		schedule: 'Program',
		feedback: 'Geri bildirim',
		onlineCounter: 'Sayaç'
	},

	/*
	 * Client options. The options panel is rendered on template generation, so
	 * these are only needed by the server.
	 * id: [label, tooltip]
	 */
	opts: {
		// Thumbnail styles
		small: 'küçük',
		sharp: 'keskin',
		hide: 'gizle',
		// Image fit modes
		none: 'yok',
		full: 'tam boyut',
		width: 'genişlemeye sığdır',
		height: 'uzunlamasına sığdır',
		both: 'ikisine göre uyarla',

		// Names for the options panel tabs
		tabs: ['Genel', 'Stil', 'ResimAra', 'Eğlence', 'Kısayollar'],
		export: [
			'Dışarı al',
			'Ayarları dosyayı al'
		],
		import: [
			'İçeri al',
			'Ayarları dosyadan al'
		],
		hidden: [
			'Gizli: 0',
			'Gizli girdileri temizle'
		],
		lang: [
			'Dil',
			'Dili değiştir'
		],
		inlinefit: [
			'Genişler',
			'İlk gönderideki resimleri genişlet ve ayarlara göre boyutlandır'
		],
		thumbs: [
			'Küçükresimler',
			'Küçük resim ayarı: '
				+ 'Küçük: 125x125, küçük dosya boyutu; '
				+ 'Keskin: 125x125, daha detaylı görüntü; '
				+ 'Gizli: tüm resimleri gizle;'
		],
		imageHover: [
			'Üstündeyken genişlet(Resim)',
			'Fare üstüne geldiğinde resimleri genişlet'
		],
		webmHover: [
			'Üstündeyken genişlet(WebM)',
			'Fare üstüne geldiğinde WebMleri genişlet. Resim ayarı açık olmalıdır'
		],
		autogif: [
			'Hareketli GIF küçükresimleri',
			'GIF küçükresimleri hareket etsin'
		],
		spoilers: [
			'Resim spoiler',
			"Resimlere spoiler koyma"
		],
		linkify: [
			'Linkleri tıklanaabilir yap',
			'Linkleri tıklanaabilir yap.'
		],
		notification: [
			'Masaüstü uyarıları',
			'Alıntı yapıldığında veya syncwatch başladığında uyarıları al'
		],
		anonymise: [
			'Anonim yap',
			'Herkesi anonim göster'
		],
		relativeTime: [
			'Saat/Zaman',
			'Zamanı 1 saat önceydi gibi göster'
		],
		nowPlaying: [
			'Şimdi Çalınan',
			'Şimdi çalınan şarkıyı göster'
		],
		// Custom localisation functions
		imageSearch: [
			function(site) {
				return lang.common.capitalize(site)  + ' Resim arama';
			},
			function(site) {
				return `Göstermek/Saklamak ${lang.common.capitalize(site)} arama bağlantıları`;
			}
		],
		illyaBGToggle: [
			'Illya Dans',
			'Arkaplanda dans eden loli'
		],
		illyaMuteToggle: [
			'Illya dans etmesin',
			'Illya dans etmesin'
		],
		horizontalPosting: [
			'Enine girdiler',
			'Enine girdiler'
		],
		replyright: [
			'[Cevapla] sağ tarafta',
			'Cevapla tuşuna sağ alta gönder'
		],
		theme: [
			'Tema',
			'Temayı seç'
		],
		userBG: [
			'Kişisel arkaplan',
			'Kişisel arkaplanı ayarla'
		],
		userBGimage: [
			'',
			"Arkaplan için resim seç"
		],
		lastn: [
			'[Son #]',
			'"Son n" ile gösterilecek girdi sayısı'
		],
		postUnloading: [
			'Girdileri bellekten çıkar',
			'Yukardan girdileri bellekten çıkararak performansı artırır'
		],
		alwaysLock: [
			'Her zaman aşağıda kal',
			'Her zaman aşağıda kal'
		],
		// Shortcut keys
		new: [
			'Yeni gönderi',
			'Yeni gönderiyi aç'
		],
		togglespoiler: [
			'Resim spoiler',
			'Spoiler ekle'
		],
		textSpoiler: [
			'Metin spoiler',
			'Spoiler ekle'
		],
		done: [
			'Bitir',
			'kapat'
		],
		expandAll: [
			'Bütün resimleri genişlet',
			'Yeni girdiler dahil bütün resimleri genişletir.'
		],
		workMode: [
			'İş modu',
			'Resimleri gizler, temayı ve arkaplanı sıfırlar'
		],
		workModeTOG: [
			'İş modu',
			'Resimleri gizler, temayı ve arkaplanı sıfırlar'
		]
	}
};

lang.common = require('./common');

module.exports = lang;
