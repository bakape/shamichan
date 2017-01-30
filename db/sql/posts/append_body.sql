update posts set body = body || $2 where id = $1
