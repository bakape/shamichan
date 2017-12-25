create or replace function insert_thread(
	subject varchar(100),
	imageCtr bigint,
	spoiler bool,
	id bigint,
	board text,
	op bigint,
	now bigint,
	body varchar(2000),
	flag char(2),
	posterID text,
	name varchar(50),
	trip char(10),
	auth varchar(20),
	ip inet,
	SHA1 char(40),
	imageName varchar(200),
	links bigint[][2],
	commands json[]
) returns void as $$
	insert into threads (
		board, id, postCtr, imageCtr, replyTime, bumpTime, subject
	)
		values (board, id, 1, imageCtr, now, now, subject);
	insert into posts (
		spoiler, id, board, op, time, body, flag, posterID,
		name, trip, auth, ip, SHA1, imageName, links, commands
	)
		values (
			spoiler, id, board, op, now, body, flag, posterID,
			name, trip, auth, ip, SHA1, imageName, links, commands
		);
$$ language sql;
