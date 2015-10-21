/*
Core database initiation and connection
 */

const async = require('async'),
	config = require('../config'),
	redisDB = require('redis'),
	r = require('rethinkdb');

const dbVersion = 2;
let rcon;

// Buffers commands, so no need for callback
function redisClient() {
	const client = redisDB.createClient({
		host: config.redis_host,
		port: config.REDIS_PORT
	});
	client.select(config.redis_database || 0);
	return client;
}
exports.redisClient = redisClient;
const redis = global.redis = redisClient();
redis.on('error', err => winston.error('Redis error:', err));

// Establish rethinkDB connection and intialize the database
function init(cb) {
	async.waterfall([
		next =>
			r.connect({
				host: config.rethink_host,
				port: config.rethinkdb_port
			}, next),
		(conn, next) => {
			rcon = global.rcon = conn;

			// Check if database exists
			r.dbList().contains('meguca').do(exists =>
				r.branch(exists, {}, r.dbCreate('meguca'))
			).run(rcon, next);
		},
		(res, next) => {
			rcon.use('meguca');

			// Create all tables at once
			createTables(['_main'].concat(config.BOARDS), next);
		},
		(res, next) =>
			r.table('_main').get('info').run(rcon, next),
		// Intialize main table or check version
		(info, next) => {
			if (info) {
				verifyVersion(info.dbVersion, 'RethinkDB');
				next(null, null);
			}
			else {
				r.table('_main').insert({id: 'info', dbVersion})
					.run(rcon, next);
			}
		},
		// Check redis version
		(res, next) =>
			redis.get('dbVersion', next),
		(version, next) => {
			if (version)
				verifyVersion(parseInt(version), 'Redis');
			next();
		}
		// Pass connection to callback
	], err => cb(err, rcon));
}
exports.init = init;

// Create tables, if they don't exist
function createTables(tables, cb) {
	r.expr(tables)
		.difference(r.tableList())
		.forEach(name => r.tableCreate(name))
		.run(rcon, cb);
}

function verifyVersion(version, dbms) {
	if (version !== dbVersion) {
		throw new Error(`Incompatible ${dbms} database version: ${version}; `
			+ 'See docs/migration.md');
	}
}
