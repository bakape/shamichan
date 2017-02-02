##v0/[doushio](https://github.com/lalcmellkmal/doushio) to v1
1. Fetch a fresh meguca directory
2. Install dependencies
3. Run `npm install`
4. Configure all files in `config/`. Make sure that all boards you had in
your previous setup are present, except for 'archive', if any.
5. Stop any running meguca servers.
6. If you care about your posts, force a redis save with `redis-cli save`. If
you don't `redis-cli flushall` and skip to step 10.
7. Backup your redis dump file (`/var/lib/redis/dump.rdb` on Linux)
8. Run `scripts/migration/0to1.js`
9. Copy over `www/{src,mid,thumb}` from your old setup to the new one
10. Start the server

##v1 to v4
TODO

##v2 to v3
Upgrades automatically by running the server. No action from the user required.

##v2/v3 to v4
1. Install PostgreSQL and set up the database as described in `installation.md`
2. Stop any running meguca servers
3. Checkout the latest v3 release
4. Run `make server`
5. Run `./meguca` and wait for the server to start accepting connections
6. Stop the server
7. Checkout the latest v4 release
8. Run `make clean all upgrade_v4`
9. Start the server
