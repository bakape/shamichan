create or replace function insert_backlink(
	id bigint,
	op bigint,
	msg bytea,
	link bigint[2]
) returns void as $$
	select update_log(op, msg);
	update posts
		set backlinks = backlinks || link
		where id = insert_backlink.id;
$$ language sql;
