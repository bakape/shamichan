create or replace function spoiler_image(
	id bigint,
	op bigint,
	msg bytea
) returns void as $$
	select update_log(op, msg);
	update posts
		set spoiler = true
		where id = spoiler_image.id;
$$ language sql;
