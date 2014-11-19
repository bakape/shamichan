(function () {

ComposerView.prototype.word_filter = function (words) {
	return words.replace(/(\w+)/g, function (orig) {
		var word = {
			feel: 'ＦＥＥＥＥＬ,ＦＥＥＥＥＥＥＥＬ',
			you: 'ＹＯＵＵＵ,ＹＯＵＵＵＵＵＵ',
			samurai: '士', sammy: '士', samu: '士',
			nipah: 'にぱー',
			filter: 'improver',
			filters: 'improves',
			filtering: 'improving',
			filtered: 'improved',
			what: '何',
			doushio: 'ＤＯＵＳＨＩＯ？',
			will: 'ｗｉｌｌ',
			can: 'ｃａｎ',
			could: 'ｃｏｕｌｄ',
			maru: '◯',
			'try': 'ＴＲＹ',
			unite: 'ＵＮＩＴＥ',
			tonight: 'ＴＯＮＩＧＨＴ',
			episode: 'ｅｐｉｓｏｄｅ',
			anime: 'ＴＶアニメ',
			'4chan': '２ｃｈ',
			april: '４月',
			fools: 'foolz',
			first: '＃１',
			'1st': '＃１',
			good: 'ＧＯＯＤ！',
			like: 'ｌｉｋｅ',
			'new': 'ｎｅｗ',
			fuck: 'ｋｕｓｏ',
			fucking: 'ｂａｋａｙａｒｏｕ',
			moon: '月',
			ika: 'イカ',
			geso: 'ｇｅｓｏ',
			someone: 'ｓｏｍｅｗａｎ　ｗａｎ　ｗａｎ',
			somewan: 'ｓｏｍｅｗａｎ　ｗａｎ　ｗａｎ',
			down: 'ｕｐ',
			up: 'ｄｏｗｎ',
			gainax: 'ＧＡＩＮＡＸ',
			yes: 'ＹＥＳ！',
			hello: 'Ｈｅｌｌｏ！',
			shipping: 'ｎｏｔ ｓｈｉｐｐｉｎｇ,ｓｔｏｐ ｓｈｉｐｐｉｎｇ,ｓｈｉｐｐｉｎｇ＆ｈａｎｄｌｉｎｇ',
			ship: 'ＳＨｉＰ,ｄｏｎ’ｔ ｓｈｉｐ,ｎｉｃｅ ｂｏａｔ,ｆｒｅｉｇｈｔ',
			mahou: 'ｍａｈｏｕ☆',
			magic: 'ｍａｈｏｕ☆',
			magical: 'ｍａｈｏｕ☆',
			normal: 'ｆｕｔｓｕｕ',
			stop: 'ｓｔｏｐ,ＳＴＡＨＰ',
			please: 'ｐｌｅａｓｅ',
			forever: 'ｚｕｔｔｏ　ｚｕｔｔｏ,ｉｔｓｕｍａｄｅｍｏ',
			weeaboo: 'ｇａｉｊｉｎ',
			weeaboos: 'ｇａｉｊｉｎ',
			niwaka: 'ｇａｉｊｉｎ',
			niwakas: 'ｇａｉｊｉｎ',
			gaijin: 'ｇａｉｊｉｎ',
			elevens: '日本人',
			eleven: '日本人',
			japan: '日本',
			lewd: 'Ｈ',
			ecchi: 'Ｈ',
			hentai: 'Ｈ',
			porn: 'Ｈ',
			mad: '［ＭＡＤ］',
			umad: 'あなた［ＭＡＤ］',
			angry: '［ＭＡＤ］',
			enrage: '［ＭＡＤ］',
			enraged: '［ＭＡＤ］',
			bothered: '［ＭＡＤ］',
			hot: 'ａｔｓｕｉ',
			today: '今日',
			whoa: 'ｗｈｏａ',
			moe: 'ｍｏｅ♥',
			touhou: '２ｈｕ',
			ready: 'Ｌ＠ＤＹ',
			me: 'ｍｉｉ',
			mii: 'ｍｉｉ',
		}[orig.toLowerCase()];
		if (word && word.indexOf(',') >= 0) {
			word = word.split(',');
			word = word[Math.floor(Math.random() * word.length)];
		}
		return (word || orig).replace('>', '<');
	});
};

var $b = $('body');
var back = $b.css('background');
$b.css('background', 'black');
setTimeout(function () { $b.css('background', back); }, 200);

})();
