create or replace function ban_post(
	id bigint,
	op bigint,
	msg bytea
) returns void as $$
	select update_log(op, msg);
	update posts
		set banned = true
		where id = ban_post.id;
$$ language sql;
