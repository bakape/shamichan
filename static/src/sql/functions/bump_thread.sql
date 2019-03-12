create function bump_thread(op bigint, bump_time bool = false) returns void
as $$
declare
	now_unix bigint := extract(epoch from now());
begin
	update threads
		set replyTime = now_unix
		where id = op;
	if bump_thread.bump_time and post_count(bump_thread.op) < 5000 then
		update threads
			set bumpTime = now_unix
			where id = bump_thread.op;
	end if;
end;
$$ language plpgsql;
