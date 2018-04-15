delete from mod_log
	where created < now() at time zone 'utc' + '-7 days'
