update posts
set body = $1
where id = $2 and editing = true
