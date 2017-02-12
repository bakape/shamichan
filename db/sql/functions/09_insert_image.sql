create or replace function insert_image(
	id bigint,
	op bigint,
	msg bytea,
	SHA1 char(40),
	name varchar(200)
) returns void as $$
	select update_log(op, msg);
	update posts
		set SHA1 = insert_image.SHA1,
			imageName = name
		where id = insert_image.id;
$$ language sql;
