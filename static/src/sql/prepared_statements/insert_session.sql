insert into sessions (account, token, expires)
values (
	$1,
	$2,
	now()
		+ (
			select (val->>'sessionExpiry' || ' days')::interval
			from main
			where id = 'config'
		)
)
