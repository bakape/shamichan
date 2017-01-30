update posts set body = left(body, -1) where id = $1
