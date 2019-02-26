create function post_count(op bigint)
returns bigint as $$
declare
	c bigint;
begin
	select count(*) into c
		from posts
		where posts.op = post_count.op;
	return c;
end;
$$ language plpgsql;
