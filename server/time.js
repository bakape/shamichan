var common = require('../common');
var hooks = require('../util/hooks');

function parse_timezone(tz) {
	if (!tz && tz != 0)
		return null;
	tz = parseInt(tz, 10);
	if (isNaN(tz) || tz < -24 || tz > 24)
		return null;
	return tz;
}

hooks.hook_sync('buildETag', function (info) {
	var tz = parse_timezone(info.req.cookies.timezone);
	if (tz) {
		info.req.tz_offset = tz;
		info.etag += '-tz' + tz;
	}
});

// Send server time to client
hooks.hook('clientSynced', function(info, cb){
	var time = new Date().getTime();
	info.client.send([0, common.GET_TIME, time]);
	cb(null);
});
