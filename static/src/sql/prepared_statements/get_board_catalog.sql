select jsonb_agg(
	get_thread(id, -6) - 'page'
	order by sticky desc, bump_time desc
)
from threads
where board = $1
