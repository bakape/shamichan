create or replace function close_post(
	id bigint,
	op bigint,
	body varchar(2000),
	links bigint[][2],
	com json[]
) returns void as $$
	update threads
		set replyTime = floor(extract(epoch from now()))
		where id = close_post.op;
	update posts
		set editing = false,
			links = close_post.links,
			commands = close_post.com,
			body = close_post.body
		where id = close_post.id;
$$ language sql;
