/*
Core database initiation and connection
 */

const async = require('async'),
	config = require('../config'),
	redisDB = require('redis'),
	rethink = global.rethink = require('rethinkdb');

const dbVersion = 2;

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

let rcon;

// Establish rethinkDB connection and intialize the database
function init(cb) {
	async.waterfall([
		next =>
			rethink.connect({
				host: config.rethink_host,
				port: config.rethinkdb_port
			}, next),
		(conn, next) => {
			rcon = global.rcon = conn;

			// Check if database exists
			rethink.dbList().contains('meguca').do(exists =>
				rethink.branch(exists,
					{dbs_created: 0},
					rethink.dbCreate('meguca'))
			).run(rcon, next);
		},
		(res, next) => {
			rcon.use('meguca');
			createTables(['main'], next);
		},
		(res, next) =>
			rethink.table('main').get('info').run(rcon, next),
		// Intialize main table or check version
		(info, next) => {
			if (info) {
				verifyVersion(info.dbVersion, 'RethinkDB');
				next(null, null);
			}
			else {
				rethink.table('main').insert([
					{id: 'info', dbVersion},
					{id: 'post_ctr'},
					{id: 'threads'}
				]).run(rcon, next);
			}
		},
		// Check redis version
		(res, next) =>
			redis.get('dbVersion', next),
		(version, next) => {
			if (version)
				verifyVersion(parseInt(version), 'Redis');
			next();
		},
		initBoards
	], cb);
}
exports.init = init;

// Create tables, if they don't exist
function createTables(tables, cb) {
	rethink.expr(tables)
		.difference(rethink.tableList())
		.forEach(name => rethink.tableCreate(name))
		.run(rcon, cb);
}

function verifyVersion(version, dbms) {
	if (version !== dbVersion) {
		throw new Error(`Incompatible ${dbms} database version: ${version} ;`
			+ 'See docs/migration.md');
	}
}

function initBoards(cb) {
	createTables(config.BOARDS.map((board => '_' + board)), err => cb(err))
}
