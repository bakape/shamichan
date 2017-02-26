create or replace function backspace(
	id bigint,
	op bigint,
	msg bytea
) returns void as $$
	select update_log(op, msg);
	update posts
		set body = substring(body from 0 for octet_length(body))
		where id = backspace.id;
$$ language sql;
