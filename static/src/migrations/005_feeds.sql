create or replace function to_unix(t timestamptz)
returns bigint
language sql stable parallel safe strict
as $$
	select extract(epoch from t)::bigint;
$$;
