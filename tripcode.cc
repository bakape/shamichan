#include <string.h>
#include <unistd.h>
#include <v8.h>

using namespace v8;

static Handle<Value> hash_callback(Arguments const &args) {
	size_t len;
	char salt[3] = "..";
	static const char *from = ":;<=>?@[\\]^_`", *to = "ABCDEFGabcdef";
	const char *c, *key;
	char *digest;
	if (args.Length() != 1)
		return Null();
	/* TODO: tweak for compatibility, Shift-JIS, etc. */
	String::AsciiValue input(args[0]);
	key = *input;
	if (!key)
		return Null();
	len = input.length();
	if (len == 1) salt[0] = 'H';
	else if (len == 2) { salt[0] = key[1]; salt[1] = 'H'; }
	else if (len) strncpy(salt, &key[1], 2);
	if (salt[0] < '.' || salt[0] > 'z')
		salt[0] = '.';
	else if ((c = strchr(from, salt[0])))
		salt[0] = to[c - from];
	if (salt[1] < '.' || salt[1] > 'z')
		salt[1] = '.';
	else if ((c = strchr(from, salt[1])))
		salt[1] = to[c - from];
	digest = crypt(key, salt);
	len = strlen(digest);
	digest[len - 11] = '!';
	return String::New(digest + len - 11);
}

extern "C" void init(Handle<Object> target) {
	target->Set(String::New("hash"),
		FunctionTemplate::New(&hash_callback)->GetFunction());
}
