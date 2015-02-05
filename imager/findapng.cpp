#include <fstream>
#include <algorithm>
#include <iterator>

#include <node.h>
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
Output: true if apng, false if png,Undefined(and throws and exception) if there is an error.*/
Handle<Value> findapngCpp(const Arguments& args)
{
  HandleScope scope;

  if(args.Length()<1){
    ThrowException(Exception::TypeError(String::New("Wrong number of arguments")));
    return scope.Close(Undefined());
  }
  if(!args[0]->IsString()){
    ThrowException(Exception::TypeError(String::New("Wrong argument (should be a filename)")));
    return scope.Close(Undefined());
  }
  String::Utf8Value filename(args[0]->ToString());
  std::ifstream in (*filename,std::ios_base::binary);
  if(!in.is_open()){
    ThrowException(Exception::Error(String::New("Can't open file")));
    return scope.Close(Undefined());
  }

  if(!isPNG(in)){
    ThrowException(Exception::Error(String::New("File isn't a png")));
    return scope.Close(Undefined());
  }

  std::istream_iterator<unsigned char> sta(in);
  std::istream_iterator<unsigned char> end;

  return scope.Close(Boolean::New((std::search(sta,end,apng.begin(),apng.end())!=end)));
}
void Init(Handle<Object> exports) {
  exports->Set(String::NewSymbol("findapngCpp"),
    FunctionTemplate::New(findapngCpp)->GetFunction());
}
NODE_MODULE(findapng,Init)
