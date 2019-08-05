select position
from staff
where account = $1 and board in ('all', $2)
order by position desc
limit 1
