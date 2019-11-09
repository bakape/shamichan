create or replace function post_board(id bigint)
returns text
language sql stable parallel safe strict
as $$
	select t.board
	from posts p
	join threads t on (t.id = p.op)
	where p.id = post_board.id;
$$;
