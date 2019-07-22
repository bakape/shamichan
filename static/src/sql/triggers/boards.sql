create or replace function after_boards_insert()
returns trigger as $$
begin

	perform pg_notify('boards.updated', new.id);

	-- Init pyu value
	insert into pyu (id, pcount) values (new.id, 0);

	return null;
end;
$$ language plpgsql;

create or replace function after_boards_update()
returns trigger as $$
begin
	perform pg_notify('boards.updated', new.id);
	return null;
end;
$$ language plpgsql;

create or replace function after_boards_delete()
returns trigger as $$
begin
	perform pg_notify('boards.updated', old.id);
	return null;
end;
$$ language plpgsql;
