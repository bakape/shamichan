select
	jsonb_agg(
		jsonb_build_object(
			'thread', t.id,
			'recent_posts', coalesce(r.val, '{}'::jsonb),
			'open_posts', coalesce(o.val, '{}'::jsonb)
		)
	)
from threads t
left join (
	select
		r.thread,
		jsonb_object_agg(
			r.id,
			to_unix(r.created_on)
		) val
	from posts r
	where r.created_on > now() - interval '16 minutes'
	group by r.thread
) r on r.thread = t.id
left join (
	select
		o.thread,
		jsonb_object_agg(
			o.id,
			jsonb_build_object(
				'has_image', o.image is not null,
				'image_spoilered', o.image_spoilered,
				'created_on', to_unix(o.created_on),
				'thread', o.thread,
				'body', o.body
			)
		) val
	from posts o
	where o.open
	group by o.thread
) o on o.thread = t.id
