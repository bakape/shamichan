create or replace function validate_tags(tags varchar(20)[])
returns bool
language plpgsql immutable parallel safe strict
as $$
declare
	tag varchar(20);
	len smallint;
begin
	foreach tag in array tags
	loop
		if lower(tag) != tag then
			raise exception 'tag not lowercase: %', tag;
		end if;
		if tag = '' then
			raise exception 'empty tag';
		end if;
	end loop;

	len = array_length(tags, 1);
	if not len between 1 and 3 then
		raise exception 'invalid tag set length: %', len;
	end if;
	if
		(
			select count(*)
			from (
				select distinct x
				from unnest(tags) x
			) _
		)
		!= len
	then
		raise exception 'tag set contains duplicates';
	end if;

	return true;
end;
$$;

alter table threads drop constraint threads_tags_check;

alter table threads
add constraint threads_tags_check
check (validate_tags(tags));
