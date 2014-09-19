var request = require('request');

(function autoJoe(){
	var again = function(){
		module.exports.isJoe = false;
		module.exports.isSpecial = false;
		module.exports.isJapanese = false;
		module.exports.isMan = false;
		setTimeout(autoJoe, 10000);
	};
	var opts = {
		url: 'https://r-a-d.io/api',
		json: true,
	};
	request.get(opts, function (err, resp, json){
		if (err)
			return again();
		if (resp.statusCode != 200)
			return again();
		if (!json || !json.main)
			return again();
		var song = json.main.np;
		module.exports.isJoe = /Girls,? Be Ambitious/ig.test(song);
		module.exports.isSpecial = /Super Special/ig.test(song);
		module.exports.isJapanese = /Turning Japanese/ig.test(song);
		module.exports.isMan = /Make a Man Out of You|Be a Man/ig.test(song);
		return setTimeout(autoJoe, 10000); 
	});
})();
