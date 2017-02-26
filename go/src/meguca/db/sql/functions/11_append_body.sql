create or replace function append_body(
	id bigint,
	op bigint,
	s varchar(3),
	msg bytea
) returns void as $$
	select update_log(op, msg);
	update posts
		set body = body || s
		where id = append_body.id;
$$ language sql;
