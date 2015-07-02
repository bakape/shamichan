#include <nan.h>
#include "mnemonizer.h"

using namespace v8;

void Init(Handle<Object> exports) {
	mnemonizer::Init(exports);
}
NODE_MODULE(mnemonics,Init)
