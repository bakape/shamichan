create or replace function close_post(
	id bigint,
	op bigint,
	msg bytea,
	links bigint[][2],
	com json[]
) returns void as $$
	select update_log(op, msg);
	update posts
		set editing = false,
			links = close_post.links,
			commands = com
		where id = close_post.id;
$$ language sql;
