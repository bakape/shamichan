create or replace function splice_body(
	id bigint,
	op bigint,
	body varchar(2000),
	msg bytea
) returns void as $$
	select update_log(op, msg);
	update posts
		set body = splice_body.body
		where id = splice_body.id;
$$ language sql;
