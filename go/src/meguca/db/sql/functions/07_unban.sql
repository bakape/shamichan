create or replace function unban(
	board varchar(3),
	id bigint,
	by varchar(20)
) returns void as $$
	delete from bans
		where board = unban.board and forPost = id;
	notify bans_updated;
	select log_moderation(1::smallint, board, id, by);
$$ language sql;
