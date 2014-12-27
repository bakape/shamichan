#!/bin/bash
# Backup to MEGA
# Depends on https://github.com/t3rm1n4l/megacmd/
# Usage: Add to crontab with either 'redis' or 'imgs' as arguments.
# Settings:
home=/home/doushio
img_folder=${home}/server/www/
megacmd=`which megacmd`
config=${home}/.megacmd.json
dump=/var/lib/redis/dump.rdb
mega_dest=meguca_backups

date=`date -u +%Y%m%d%H%M%S`
if [[ "$@" = "redis" ]]; then
	redis-cli save
	archive=/tmp/meguca-redis-backup-${date}.tgz
	src=$dump
elif [[ "$@" = "imgs" ]]; then
	archive=/tmp/meguca-imgs-backup-${date}.tgz
	src="${img_folder}{mid,src,thumb,vint}"
else 
	echo "U WOT M8?"
fi
tar cfpz $archive $src
$megacmd -conf=$config put $archive mega:/${mega_dest}/`basename $archive`
