create or replace function insert_post(
	editing bool,
	spoiler bool,
	id bigint,
	board varchar(3),
	op bigint,
	now bigint,
	body varchar(2000),
	name varchar(50),
	trip char(10),
	auth varchar(20),
	password bytea,
	ip inet,
	SHA1 char(40),
	imageName varchar(200),
	links bigint[][2],
	backlinks bigint[][2],
	commands json[],
	sage bool
) returns void as $$
	select bump_thread(op, not sage, SHA1 is not null);
	insert into posts (
		editing, spoiler, id, board, op, time, body, name, trip, auth, password,
		ip, SHA1, imageName, links, backlinks, commands, sage
	)
		values (
			editing, spoiler, id, board, op, now, body, name, trip, auth,
			password, ip, SHA1, imageName, links, backlinks, commands, sage
		);
$$ language sql;
