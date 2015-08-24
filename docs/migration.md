##v0/[doushio](https://github.com/lalcmellkmal/doushio) to v1
1. Fetch a fresh meguca directory
2. Install dependancies
3. Run `npm install`
4. Configure all files in `config/`. Make sure that all boards you had in 
your previous setup are present, except for 'archive', if any.
5. Stop any running meguca servers. 
6. If you care about your posts, force a redis save with `redis-cli save`. If 
you don't `redis-cli flushall` and skip to step 10.
7. Backup your redis dump file (`/var/lib/redis/dubm.rdb` on Linux)
8. Run `scripts/migration_0to1.js`
9. Copy over `www/{src,mid,thum}` from your old setup to the new one
10. Start the server
