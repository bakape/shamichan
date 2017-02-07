create or replace function update_log (id bigint, msg bytea) returns void as $$
	update threads
		set log = array_append(log, msg)
		where id = update_log.id;
	select pg_notify('t:' || id, '');
$$ language sql;
