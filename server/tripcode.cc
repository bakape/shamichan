#define _XOPEN_SOURCE
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <v8.h>

using namespace v8;

static char SECURE_SALT[21] = "$5$";

static int load_secure_salt(void) {
	FILE *fp = popen("node config.js --show-config SECURE_SALT", "r");
	if (!fp) {
		perror("popen");
		return 0;
	}
	if (fread(SECURE_SALT + 3, 1, 16, fp) != 16) {
		fprintf(stderr, "Invalid SECURE_SALT.\n");
		pclose(fp);
		return 0;
	}
	pclose(fp);
	SECURE_SALT[19] = '$';
	SECURE_SALT[20] = 0;
	return 1;
}

static void fix_char(char &c) {
	static const char *from = ":;<=>?@[\\]^_`", *to = "ABCDEFGabcdef";
	const char *p;
	if (c < '.' || c > 'z')
		c = '.';
	else if ((p = strchr(from, c)))
		c = to[p - from];
}

static void hash_trip(const char *key, size_t len, char *dest) {
	char *digest, salt[3] = "..";
	if (len == 1)
		salt[0] = 'H';
	else if (len == 2) {
		salt[0] = key[1];
		salt[1] = 'H';
	}
	else if (len)
		strncpy(salt, key + 1, 2);
	fix_char(salt[0]);
	fix_char(salt[1]);
	digest = crypt(key, salt);
	if (!digest)
		return;
	len = strlen(digest);
	if (len < 11)
		return;
	digest += len - 11;
	digest[0] = '!';
	strncpy(dest, digest, 12);
}

static void hash_secure(char *key, size_t len, char *dest) {
	size_t i;
	char buf[21] = "$5$", *digest;
	if (len > 128) {
		len = 128;
		key[128] = 0;
	}
	for (i = 0; i < len; i++)
		fix_char(key[i]);
	digest = crypt(key, SECURE_SALT);
	if (!digest)
		return;
	len = strlen(digest);
	if (len < 12)
		return;
	digest += len - 12;
	digest[0] = digest[1] = '!';
	strncpy(dest, digest, 13);
}

static Handle<Value> hash_callback(Arguments const &args) {
	if (args.Length() != 2)
		return Null();
	String::AsciiValue trip(args[0]->ToString()),
		secure(args[1]->ToString());
	char digest[24];
	digest[0] = 0;
	if (*trip && trip.length())
		hash_trip(*trip, trip.length(), digest);
	if (*secure && secure.length())
		hash_secure(*secure, secure.length(), digest + strlen(digest));
	return String::New(digest);
}

extern "C" void init(Handle<Object> target) {
	if (load_secure_salt())
		target->Set(String::New("hash"),
			FunctionTemplate::New(&hash_callback)->GetFunction());
}
