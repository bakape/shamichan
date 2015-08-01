var events = require('events'),
    util = require('util'),
    winston = require('winston');

var JOB_LIMIT = 1;
var JOB_TIMEOUT = 30 * 1000;

var JOB_QUEUE = [];
var JOBS_RUNNING = 0;

function schedule(job, cb) {
	if (job && job.jobRunning)
		winston.warn("Job "+job.describe_job()+" already running!");
	else if (job && JOB_QUEUE.indexOf(job) >= 0)
		winston.warn("Job "+job.describe_job()+" already scheduled!");
	else if (job) {
		JOB_QUEUE.push(job);
		if (cb) {
			/* Sucks */
			job.once('finish', cb);
			job.once('timeout', cb.bind(null, "Timed out."));
		}
	}

	while (JOB_QUEUE.length && JOBS_RUNNING < JOB_LIMIT)
		JOB_QUEUE.shift().start_job();
}
exports.schedule = schedule;

function Job() {
	events.EventEmitter.call(this);
}
util.inherits(Job, events.EventEmitter);
exports.Job = Job;

Job.prototype.start_job = function () {
	if (this.jobRunning) {
		winston.warn(this.describe_job() + " already started!");
		return;
	}
	JOBS_RUNNING++;
	this.jobRunning = true;
	this.jobTimeout = setTimeout(this.timeout_job.bind(this), JOB_TIMEOUT);
	setTimeout(this.perform_job.bind(this), 0);
};

Job.prototype.finish_job = function (p1, p2) {
	if (!this.jobRunning) {
		winston.warn("Attempted to finish stopped job: "
				+ this.describe_job());
		return;
	}
	clearTimeout(this.jobTimeout);
	this.jobTimeout = 0;
	this.jobRunning = false;
	JOBS_RUNNING--;
	if (JOBS_RUNNING < 0)
		winston.warn("Negative job count: " + JOBS_RUNNING);
	/* use `arguments` later */
	this.emit('finish', p1, p2);
	schedule(null);
};

Job.prototype.timeout_job = function () {
	var desc = this.describe_job();
	if (!this.jobRunning) {
		winston.warn("Job " + desc + " timed out though finished?!");
		return;
	}

	winston.error(desc + " timed out.");

	this.jobTimeout = 0;
	this.jobRunning = false;
	JOBS_RUNNING--;
	if (JOBS_RUNNING < 0)
		winston.warn("Negative job count: " + JOBS_RUNNING);

	this.emit('timeout');
	schedule(null);
};

Job.prototype.describe_job = function () {
	return "<anonymous job>";
};
