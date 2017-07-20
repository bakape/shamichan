delete from reports
	where created < now() + '-7 days'
