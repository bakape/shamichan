/* Dumps the redis DB to rdb, then uploads that to S3.
 * Configure below, and provide upkeep/credentials.json.
 */

var AWS = require('aws-sdk'),
    config = require('../config'),
    crypto = require('crypto'),
    db = require('../db'),
    fs = require('fs');

var BUCKET = 'backupbucket';
var ACL = 'bucket-owner-full-control';
var RDB_PATH = '/Users/lalc/dump.rdb';
var BGSAVE_TIMEOUT = 5*60;

var r, s3;

function dump_rdb(cb) {
	console.log('Dumping rdb...');
	r.lastsave(function (err, lastSave) {
		if (err) return err;
		r.bgsave(function (err)	{
			if (err) return err;
			setTimeout(poll_for_lastsave_after.bind(null,
					lastSave, 0, cb), 200);
		});
	});
}

function poll_for_lastsave_after(lastSave, seconds, cb) {
	// Will take slightly longer than BGSAVE_TIMEOUT seconds since
	// we're waiting for a lastsave command between each second
	if (seconds > BGSAVE_TIMEOUT)
		return cb("bgsave timeout");

	r.lastsave(function (err, newSave) {
		if (err)
			return cb(err);
		if (newSave > lastSave) {
			console.log('Obtained dump after', seconds, 'sec');
			check_mtime(RDB_PATH, newSave, cb);
			return;
		}
		if (newSave < lastSave)
			return cb("Time travel");
		setTimeout(poll_for_lastsave_after.bind(null,
				lastSave, seconds+1, cb), 1000);
	});
}

function check_mtime(path, lastsave, cb) {
	console.log('  redis lastsave:', lastsave);
	fs.stat(path, function (err, stat) {
		if (err)
			return cb(err);
		var mtime = Math.floor(stat.mtime.getTime() / 1000);
		console.log('  ' + path + ' mtime:', mtime);
		if (Math.abs(mtime - lastsave) > 1000)
			return cb("Bad mtime: "+lastsave+" != "+mtime);
		cb(null);
	});
}

function upload(kind, filename, cb) {
	var now = new Date();
	var random = crypto.randomBytes(8).toString('hex');
	var dir = now.getUTCFullYear() + '-' + pad2(now.getUTCMonth()+1);
	var name = dir+'-'+pad2(now.getUTCDate()) + '-' + kind + '-' + random;
	var key = dir + '/' + name;

	var stream = fs.createReadStream(filename);
	var params = {
		ACL: ACL,
		Bucket: BUCKET,
		Key: key,
		Body: stream,
	};

	process.stdout.write('Writing s3://'+BUCKET+'/'+key+' ...');
	s3.putObject(params, function (err, data) {
		if (err) {
			console.log('failed!');
			return cb(err);
		}
		console.log('done.');
		console.log('ETag:', data.ETag);
		cb(null);
	});
}

function pad2(n) {
	return (n < 10 ? '0' : '') + n;
}

if (require.main == module) {
	s3 = new AWS.S3().client;
	s3.config.loadFromPath('scripts/credentials.json');
	r = db.redis_client();

	dump_rdb(function (err) {
		if (err)
			throw err;
		upload('dump.rdb', RDB_PATH, function (err) {
			if (err)
				throw err;
			process.exit();
		});
	});
}
