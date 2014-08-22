var db = require('../db'),
	config = require('../config'),
	cronJob = require('cron').CronJob;

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
	var date = new Date().getDate();
	db.ah_set(date, ah.join(), ah_check);
}

// Check if the current hour is an anon hour
function ah_check(){
	db.ah_get(
		function(err, read){
			var hours = new Date().getHours();
			module.exports.anon_hour = (read.hours.split(',').indexOf(String(hours)) > -1);
		}
	);
}

