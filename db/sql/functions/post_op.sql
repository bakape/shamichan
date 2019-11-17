create or replace function post_op(id bigint)
returns bigint
language sql stable parallel safe strict
as $$
	select op
	from posts
	where id = post_op.id;
$$;
