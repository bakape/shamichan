#include "apngDetector.h"

using namespace v8;

const std::string apng("acTL");
const std::string idat("IDAT");

Nan::Persistent<Function> apngDetector::constructor;

apngDetector::apngDetector(){
  cOffset = 8;//skip signature
}
apngDetector::~apngDetector(){

}
NAN_MODULE_INIT(apngDetector::Init){

  Local<FunctionTemplate> tpl = Nan::New<FunctionTemplate>(New);
  tpl->SetClassName(Nan::New<String>("apngDetector").ToLocalChecked());
  tpl->InstanceTemplate()->SetInternalFieldCount(2);

  Nan::SetPrototypeMethod(tpl,"Detect",Detect);

  constructor.Reset(tpl->GetFunction());
  Nan::Set(target,
	   Nan::New<String>("apngDetector").ToLocalChecked(),
	   tpl->GetFunction());
}
NAN_METHOD(apngDetector::New){
  if(info.IsConstructCall()){
    apngDetector* obj = new apngDetector();
    obj->Wrap(info.This());
    info.GetReturnValue().Set(info.This());
  }else {
    Local<Function> cons = Nan::New<Function>(constructor);
    info.GetReturnValue().Set(cons->NewInstance());
  }
}
/* Gets a buffer with image data in it.
 * returns 0, if it doesn't find anything.
 * returns 1  if the file is an apng
 * returns 2  if the file isn't an apng
 */
NAN_METHOD(apngDetector::Detect){
  apngDetector* obj = ObjectWrap::Unwrap<apngDetector>(info.Holder());

  if(!info[0]->IsObject())
    return Nan::ThrowTypeError("apngDetector:Wrong argument (did you pass a buffer?)");
  char* buffer = node::Buffer::Data(info[0]);
  uint length = node::Buffer::Length(info[0]);
  uint offset = 0;
  int result = 0;

  int carryOffset = obj->cOffset;
  if(carryOffset<0){ //carrying split chunk type from previous pipe
    std::string newBuf = obj->cBytes+buffer;
    result = obj->checkChunk((unsigned char*)newBuf.c_str(),offset);
    if(result){
      info.GetReturnValue().Set(result);
      return;
    }
  }else
    offset+=carryOffset; //carrying previous offset

  while((offset+8)<length){
    result = obj->checkChunk((unsigned char*)buffer,offset);
    if(result){
      info.GetReturnValue().Set(result);
      return;
    }
  }
  //didn't find anything+
  obj->cOffset = offset-length;
  if(offset<length){ //Split chunk length/type
    std::string carry(buffer+offset,buffer+length);
    obj->cBytes = carry;
  }
  info.GetReturnValue().Set(0);
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
