var config = require('../config').AUTOJOE_CONF,
	request = require('request');

(function autoJoe(){
	// Query r/a/dio API
	request.get({url: 'https://r-a-d.io/api', json: true,}, function (err, resp, json){
		if (err || resp.statusCode != 200 || !json || !json.main){
			module.exports.isJoe = false;
			return setTimeout(autoJoe, 10000);
		}
		var song = json.main.np;
		var isJoe = false;
		for (i = 0; i < config.length; i++){
			var pat = config[i].pattern;
			if (!pat.test(song))
				continue;
			isJoe = true;
			module.exports.name = config[i].name;
			break;
		}
		module.exports.isJoe = isJoe;
		return setTimeout(autoJoe, 10000); 
	});
})();
