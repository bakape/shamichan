#include "apngDetector.h"

using namespace v8;

const std::string apng("acTL");
const std::string idat("IDAT");

Persistent<Function> apngDetector::constructor;

apngDetector::apngDetector(){
  cOffset = 8;//skip signature
}
apngDetector::~apngDetector(){

}
void apngDetector::Init(Handle<Object> exports) {
  NanScope();

  Local<FunctionTemplate> tpl = NanNew<FunctionTemplate>(New);
  tpl->SetClassName(NanNew("apngDetector"));
  tpl->InstanceTemplate()->SetInternalFieldCount(2);

  NODE_SET_PROTOTYPE_METHOD(tpl,"Detect",Detect);

  NanAssignPersistent(constructor,tpl->GetFunction());
  exports->Set(NanNew("apngDetector"),tpl->GetFunction());
}
NAN_METHOD(apngDetector::New){
  NanScope();

  if(args.IsConstructCall()){
    apngDetector* obj = new apngDetector();
    obj->Wrap(args.This());
    NanReturnValue(args.This());
  }else {
    Local<Function> cons = NanNew<Function>(constructor);
    NanReturnValue(cons->NewInstance());
  }
}
/* Gets a buffer with image data in it.
 * returns 0, if it doesn't find anything.
 * returns 1  if the file is an apng
 * returns 2  if the file isn't an apng
 */
NAN_METHOD(apngDetector::Detect){
  NanScope();
  apngDetector* obj = ObjectWrap::Unwrap<apngDetector>(args.Holder());

  if(!args[0]->IsObject())
    return NanThrowTypeError("apngDetector:Wrong argument (did you pass a buffer?)");
  char* buffer = node::Buffer::Data(args[0]);
  uint length = node::Buffer::Length(args[0]);
  uint offset = 0;
  int result = 0;

  int carryOffset = obj->cOffset;
  if(carryOffset<0){ //carrying split chunk type from previous pipe
    std::string newBuf = obj->cBytes+buffer;
    result = obj->checkChunk((unsigned char*)newBuf.c_str(),offset);
    if(result)
      NanReturnValue(NanNew<Number>(result));
  }else
    offset+=carryOffset; //carrying previous offset

  while((offset+8)<length){
    result = obj->checkChunk((unsigned char*)buffer,offset);
    if(result)
      NanReturnValue(NanNew<Number>(result));
  }
  //didn't find anything+
  obj->cOffset = offset-length;
  if(offset<length){ //Split chunk length/type
    std::string carry(buffer+offset,buffer+length);
    obj->cBytes = carry;
  }
  NanReturnValue(NanNew<Number>(0));
}
int apngDetector::checkChunk(const unsigned char* buffer,uint& offset){
  std::string chunkID(buffer+offset+4,buffer+offset+8);
  if(chunkID==apng)
      return 1;
  if(chunkID==idat)
      return 2;
  uint chunkLen =  (*(buffer+offset)<<24)|
                  (*(buffer+offset+1)<<16)|
                  (*(buffer+offset+2)<<8)|
                  (*(buffer+offset+3)); //read always in BE order
  offset+=chunkLen+12;//type(4),len(4),crc(4)
  return 0;
}
