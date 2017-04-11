create or replace function insert_image(
	id bigint,
	op bigint,
	SHA1 char(40),
	name varchar(200)
) returns void as $$
	update threads
		set imageCtr = imageCtr + 1
		where id = insert_image.op;
	update posts
		set SHA1 = insert_image.SHA1,
			imageName = insert_image.name
		where id = insert_image.id;
$$ language sql;
