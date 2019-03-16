create or replace function on_boards_insert()
returns trigger as $$
begin
	perform pg_notify('board_updated', new.id);

	-- Init pyu value
	insert into pyu (id, pcount) values (new.id, 0);

	return null;
end;
$$ language plpgsql;

create or replace function on_boards_update()
returns trigger as $$
begin
	perform pg_notify('board_updated', new.id);
	return null;
end;
$$ language plpgsql;

create or replace function on_boards_delete()
returns trigger as $$
begin
	perform pg_notify('board_updated', old.id);
	return null;
end;
$$ language plpgsql;
