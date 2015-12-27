#include <nan.h>
#include "apngDetector.h"

using namespace v8;

void Init(Handle<Object> exports) {
	apngDetector::Init(exports);
}
NODE_MODULE(findapng,Init)
