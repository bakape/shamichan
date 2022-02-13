create or replace function after_ban_insert()
returns trigger as $$
begin
    delete from last_solved_captchas where ip = new.ip;
	return null;
end;
$$ language plpgsql;