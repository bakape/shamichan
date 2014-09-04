var db = require('../db'),
	config = require('../config'),
	common = require('../common'),
	cronJob = require('cron').CronJob,
	tripcode = require('./../tripcode/tripcode');

exports.ah_init = function(){
	db.ah_get(
		function(err, read){
			// First time execution
			if (read == null)
				ah_gen();
			else{
				var date = new Date().getDate();
				// Generate a new set of anon hours on a new day
				if (read.date == date)
					ah_check();
				else 
					ah_gen();
			}
			// Check if a new hour is anon hour
			var hourly = new cronJob('0 0 1-23 * * *', ah_check, null, false);
			// Regenerate set at the start of a new day
			var daily = new cronJob('0 0 0 * * *', ah_gen, null, false);
			hourly.start();
			daily.start();
		}
	);
};

// Generate a new set of anon hours
function ah_gen(){
	var sections = config.ANON_HOURS_PER_DAY,
		ah = [],
		s = 24 / sections;
	for (i = 0; i < sections; i++){
		var m = (i * s),
			p = Math.floor(Math.random() * s);
		ah.push(m + p);
	}
	var d = new Date();
	var date = d.getDate();
	var month = d.getMonth();
	db.ah_set(date, ah.join(), month, ah_check);
}

// Check if the current hour is an anon hour
var nameDB;
function ah_check(){
	db.ah_get(
		function(err, read){
			var d = new Date();
			var hours = d.getHours();
			var month = d.getMonth();
			var anon_hour = (read.hours.split(',').indexOf(String(hours)) > -1);
			module.exports.anon_hour = anon_hour;
						
			var random_name_hour = false;
			if (anon_hour){
				// Roll for anon hour becoming a random name hour
				var chance = config.RANDOM_NAME_HOURS;
				if (chance < 10){
					for (i = 1; i < chance; i++){
						if (Math.random() > 0.9)
							random_name_hour = true;
					}
				} else
					random_name_hour = true;
			}
			// Clear the used name set at the start of a new month
			if (read.month != month){
				global.redis.del('nameDB');
				module.exports.random_name_hour = false;
			} else {
				// Load used name set from redis
				db.nameDB_get(
					function(err, res){
						if (err || !res)
							random_name_hour = false;
						nameDB = res;
						module.exports.random_name_hour = random_name_hour;
					}
				);
			}
		}
	);
}

// Pick a random name + tripcode from the currently loaded posted name set
exports.random_name = function(post){
	var combined = nameDB[Math.floor(Math.random() * nameDB.length)];
	var name = /^(.+?)\|\|\|.*?$/.exec(combined);
	var trip = /^.*?\|\|\|(.+?)$/.exec(combined);
	if (name)
		post.name = name[1];
	if (trip)
		post.trip = trip[1];
};

// Parse msg.name and write to used name set
exports.name_parse = function(msg){
	var parsed = common.parse_name(msg);
	var trip = '';
	if (parsed[1] || parsed[2])
		trip = tripcode.hash(parsed[1], parsed[2]);
	var combined = parsed[0] + '|||' + trip;
	db.nameDB_add(combined);
};
