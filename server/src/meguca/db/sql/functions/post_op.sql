create function post_op(id bigint)
returns bigint as $$
declare
	op bigint;
begin
	select op into op
		from posts
		where id = post_op.id;
	return op;
end;
$$ language plpgsql;
