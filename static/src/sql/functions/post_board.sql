create function post_board(id bigint)
returns text as $$
declare
	board text;
begin
	select t.board into board
		from posts p
		join threads t on (t.id = p.op)
		where p.id = post_board.id;
	return board;
end;
$$ language plpgsql;
