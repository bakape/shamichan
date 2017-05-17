delete from mod_log
	where created < now() + '-7 days'
