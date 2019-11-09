create or replace function thread_board(id bigint)
returns text
language sql stable parallel safe strict
as $$
	select board from threads where id = thread_board.id;
$$;
