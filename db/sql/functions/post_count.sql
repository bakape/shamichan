create or replace function post_count(op bigint)
returns bigint
language sql stable parallel safe strict
as $$
	select count(*)
	from posts
	where posts.op = post_count.op;
$$;
