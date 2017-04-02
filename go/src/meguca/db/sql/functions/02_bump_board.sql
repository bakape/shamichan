create or replace function bump_board(id varchar(3)) returns void as $$
	update boards
		set ctr = ctr + 1
		where id = bump_board.id;
$$ language sql;
