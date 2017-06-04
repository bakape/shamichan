create or replace function log_moderation(
	type smallint,
	board varchar(3),
	id bigint,
	by varchar(20)
) returns void as $$
	insert into mod_log (type, board, id, by)
		values(type, board, id, by);
$$ language sql;
