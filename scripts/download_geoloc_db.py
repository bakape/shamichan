#! /bin/env python3

import requests
from datetime import date

today = date.today()
y = today.year
m = today.month

while True:
    print("trying to download DB for year=%d month=%02d" % (y, m))
    res = requests.get(
        "https://download.db-ip.com/free/dbip-city-lite-%d-%02d.mmdb.gz" %
        (y, m),
    )
    if res.status_code == 404:
        m -= 1
        if m == 0:
            y -= 1
            m = 12
        continue
    res.raise_for_status()
    with open("dbip-city-lite.mmdb", "wb") as f:
        f.write(res.content)
        print("done")
        break
