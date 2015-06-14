var common = require('../common');
var hooks = require('../util/hooks');

// Send server time to client
hooks.hook('clientSynced', function(info, cb){
	var time = Date.now();
	info.client.send([0, common.GET_TIME, time]);
	cb(null);
});
