create or replace function is_deleted(id bigint)
returns bool
language sql stable parallel safe strict
as $$
	select exists (
		select
		from post_moderation pm
		where pm.post_id = is_deleted.id
			and pm.type = 'delete_post'
	);
$$;
