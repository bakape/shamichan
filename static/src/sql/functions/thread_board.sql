create or replace function thread_board(id bigint)
returns text as $$
	select board from threads where id = thread_board.id;
$$ language sql;
