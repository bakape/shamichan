-- Assert account can perform moderation action
create function assert_can_perform(account text, board text, level smallint)
returns void as $$
declare
	can bool;
begin
	select exists (select 1
					from staff s
					where s.board in ('all', assert_can_perform.board)
						and s.position >= assert_can_perform.level
						and s.account = assert_can_perform.account)
		into can;
	if not can then
		raise exception 'access denied';
	end if;
end;
$$ language plpgsql;
