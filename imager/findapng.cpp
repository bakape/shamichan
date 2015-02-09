#include <fstream>
#include <algorithm>
#include <iterator>

#include <node.h>
#include <nan.h>

using namespace v8;
/*Checks the PNG magic number, returns false if it doesn't correspond to a png
or it is too short*/
const std::string apng("acTL");
const std::string magicNum("\211PNG\r\n\032\n");
bool isPNG(std::ifstream &in)
{
  char buf[8];
  in.read(buf,8);
  return (buf==magicNum);
}
/*Checks if a file is png or apng.
Input: filename
Output: true if apng, false if png, -1if file isn't a png or apng and Undefined(and throws and exception) if there is an error.*/
NAN_METHOD(findapngCpp)
{
  NanScope();

  if(args.Length()<1){
    NanThrowTypeError("Wrong number of arguments");
    NanReturnUndefined();
  }
  if(!args[0]->IsString()){
    NanThrowTypeError("Wrong argument (should be a filename)");
    NanReturnUndefined();
  }
  std::ifstream in (*NanUtf8String(args[0]),std::ios_base::binary);
  if(!in.is_open()){
    NanThrowError("Can't open file");
    NanReturnUndefined();
  }

  if(!isPNG(in))
    NanReturnValue(NanNew<Number>(-1));

  std::istream_iterator<unsigned char> sta(in);
  std::istream_iterator<unsigned char> end;

  NanReturnValue(NanNew<Boolean>(std::search(sta,end,apng.begin(),apng.end())!=end));
}
void Init(Handle<Object> exports) {
  exports->Set(NanNew<String>("findapngCpp"),
  NanNew<FunctionTemplate>(findapngCpp)->GetFunction());
}
NODE_MODULE(findapng,Init)
