-- Assert account can perform moderation action
create or replace function assert_can_perform(
	account text,
	board text,
	level smallint
)
returns void
language plpgsql stable parallel safe strict
as $$
declare
	can bool;
begin
	select into can exists (
		select
		from staff s
		where s.board in ('all', assert_can_perform.board)
			and s.position >= assert_can_perform.level
			and s.account = assert_can_perform.account
	);
	if not can then
		raise exception 'access denied';
	end if;
end;
$$;
