create table main (
	id text primary key,
	val text not null
);
insert into main (id, val) values
	('version', %d),
	('config', '%s'),
	('pyu', '0');

create table accounts (
	id varchar(20) primary key,
	password bytea not null
);

create table sessions (
	account varchar(20) not null references accounts on delete cascade,
	token text not null,
	expires timestamp not null,
	primary key (account, token)
);

create table bans (
	board text not null,
	ip inet not null,
	forPost bigint default 0,
	by varchar(20) not null,
	reason text not null,
	expires timestamp not null,
	primary key (ip, board)
);

create table mod_log (
	type smallint not null,
	board text not null,
	id bigint not null,
	by varchar(20) not null,
	created timestamp default (now() at time zone 'utc')
);
create index mod_log_board on mod_log (board);
create index mod_log_created on mod_log (created);

create table images (
	apng boolean not null,
	audio boolean not null,
	video boolean not null,
	fileType smallint not null,
	thumbType smallint not null,
	dims smallint[4] not null,
	length int not null,
	size int not null,
	MD5 char(22) not null,
	SHA1 char(40) primary key,
	Title varchar(200) not null,
	Artist varchar(100) not null
);

create table image_tokens (
	token char(86) not null primary key,
	SHA1 char(40) not null references images on delete cascade,
	expires timestamp not null
);

create table boards (
	readOnly boolean not null,
	textOnly boolean not null,
	forcedAnon boolean not null,
	disableRobots boolean default false,
	flags boolean default false,
	NSFW boolean default false,
	nonLive bool default false,
	posterIDs bool default false,
	id text primary key,
	created timestamp not null,
	defaultCSS text not null,
	title varchar(100) not null,
	notice varchar(500) not null,
	rules varchar(5000) not null,
	js varchar(5000) default '',
	eightball text[] not null
);

create table staff (
	board text not null references boards on delete cascade,
	account varchar(20) not null references accounts on delete cascade,
	position varchar(50) not null
);
create index staff_board on staff (board);
create index staff_account on staff (account);

create table banners (
	board text not null references boards on delete cascade,
	id smallint not null,
	data bytea not null,
	mime text not null
);

create table loading_animations (
	board text primary key references boards on delete cascade,
	data bytea not null,
	mime text not null
);

create sequence post_id;

create table threads (
	sticky boolean default false,
	nonLive bool default false,
	board text not null references boards on delete cascade,
	id bigint primary key,
	postCtr bigint not null,
	imageCtr bigint not null,
	bumpTime bigint not null,
	replyTime bigint not null,
	subject varchar(100) not null
);
create index threads_board on threads (board);
create index bumpTime on threads (bumpTime);
create index replyTime on threads (replyTime);
create index sticky on threads (sticky);

create table posts (
	editing boolean not null,
	spoiler boolean,
	deleted boolean,
	banned boolean,
	sage boolean,
	locked boolean default false,
	id bigint primary key,
	op bigint not null references threads on delete cascade,
	time bigint not null,
	board text not null,
	flag char(2),
	posterID text,
	trip char(10),
	auth varchar(20),
	SHA1 char(40) references images on delete set null,
	name varchar(50),
	imageName varchar(200),
	body varchar(2000) not null,
	password bytea,
	ip inet,
	links bigint[][2],
	commands json[]
);
create index op on posts (op);
create index image on posts (SHA1);
create index editing on posts (editing);
create index ip on posts (ip);

create table reports (
	id bigserial primary key,
	target bigint not null,
	board text not null,
	reason text not null,
	by inet not null,
	illegal boolean not null,
	created timestamp default (now() at time zone 'utc')
);
create index report_board on reports (board);
create index report_created on reports (created);
