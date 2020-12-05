create type pending_image_status as enum (
	'pending',
	'successful',
	'failed'
);

create table pending_images (
	post bigint primary key references posts on delete cascade,

	expires timestamptz not null default now() + interval '5 minutes',

	status pending_image_status not null default 'pending',
	error text,
	constraint error_null_validity check (
		case status
			when 'failed' then error is not null
			else error is null
		end
	)

	image references images,
	constraint image_null_validity check (
		case status
			when 'successful' then image is not null
			else image is null
		end
	),

	source bytea not null,
	size int not null generated always as (octet_length(source)) stored
)
inherits (expiries);
create index pending_images_expires_idx on pending_images (expires);

create or replace function notify_pending_image_status_change()
returns trigger
language plpgsql stable parallel safe strict
as $$
begin
	perform pg_notify(
		'pending_images.status_change',
		new.post || ':' || new.status
	);
	return new;
end;
$$;

create trigger notify_pending_image_status_change
after insert or update on pending_images
for each row
execute function notify_pending_image_status_change();
