create or replace function is_deleted(id bigint)
returns bool as $$
declare
	deleted bool;
begin
	select exists (select 1
					from post_moderation pm
					where pm.post_id = is_deleted.id
						and pm.type = 2)
		into deleted;
	return deleted;
end;
$$ language plpgsql;
