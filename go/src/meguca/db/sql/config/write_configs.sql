update main
	set val = $1
	where id = 'config'
	returning pg_notify('config_updates', $1)
