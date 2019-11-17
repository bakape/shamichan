-- Returns posts with the same IP and on the same thread as the target post
create or replace function get_same_ip_posts(id bigint, account text)
returns jsonb
language plpgsql stable parallel safe strict
as $$
declare
	data jsonb;

	post_ip inet;
	op bigint;
begin
	select p.ip, p.op into post_ip, op
		from posts p
		where p.id = get_same_ip_posts.id;
	if post_ip is null then
		return null;
	end if;

	perform call bump_thread(account, target_board, 1::smallint);

	insert into mod_log (type, board, post_id, "by")
		values (7, target_board, id, account);

	select into data
		jsonb_agg(
			encode_post(p)
				|| jsonb_build_object(
					'op', p.op
				)
			order by p.id
		)
		from posts p
		join threads t on p.op = t.id
		where p.ip = post_ip and t.board = target_board;
	return data;
end;
$$;
