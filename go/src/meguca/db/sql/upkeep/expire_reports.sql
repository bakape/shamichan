delete from reports
	where created < now() at time zone 'utc' + '-7 days'
