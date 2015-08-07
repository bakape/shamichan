 /*
  1) Receive an integer for the minimal amount of bits needed to not trigger a duplicate, a 43 character base64 string and an array of  '<post number>:<43 char base64 string>' from node
  2) Convert second argument to binary
  3) loop through the array by doing the following:
    a) slice of array member's '<post number>:'
    b) convert base64 to binary
    c) XOR the first string against the array member string
    d) If the number of different bits is below the first argument return matching post number and exit. Otherwise continue to the next array member
  6) all array elements pass, return 0, if there is an error return -1
 */
#include <cstdlib>
#include <string>
#include <bitset>

#include <node.h>
#include <nan.h>

#include "compare.h"

using namespace v8;

const static TCHAR padCharacter = TEXT('=');
std::vector<BYTE> base64Decode(const std::basic_string<TCHAR>& input)
{
  size_t padding = 0;
  if (input.length())
  {
    if (input[input.length()-1] == padCharacter)
      padding++;
    if (input[input.length()-2] == padCharacter)
      padding++;
  }
  //Setup a vector to hold the result
  std::vector<BYTE> decodedBytes;
  decodedBytes.reserve(((input.length()/4)*3) - padding);
  DWORD temp=0; //Holds decoded quanta
  std::basic_string<TCHAR>::const_iterator cursor = input.begin();
  while (cursor < input.end())
  {
    for (size_t quantumPosition = 0; quantumPosition < 4; quantumPosition++)
    {
      temp <<= 6;
      if       (*cursor >= 0x41 && *cursor <= 0x5A) // This area will need tweaking if
        temp |= *cursor - 0x41;		              // you are using an alternate alphabet
      else if  (*cursor >= 0x61 && *cursor <= 0x7A)
        temp |= *cursor - 0x47;
      else if  (*cursor >= 0x30 && *cursor <= 0x39)
        temp |= *cursor + 0x04;
      else if  (*cursor == 0x2B)
        temp |= 0x3E; //change to 0x2D for URL alphabet
      else if  (*cursor == 0x2F)
        temp |= 0x3F; //change to 0x5F for URL alphabet
      else if  (*cursor == padCharacter) //pad
      {
        switch( input.end() - cursor )
        {
        case 1: //One pad character
          decodedBytes.push_back((temp >> 16) & 0x000000FF);
          decodedBytes.push_back((temp >> 8 ) & 0x000000FF);
          return decodedBytes;
        case 2: //Two pad characters
          decodedBytes.push_back((temp >> 10) & 0x000000FF);
          return decodedBytes;
        }
      }
      cursor++;
    }
    decodedBytes.push_back((temp >> 16) & 0x000000FF);
    decodedBytes.push_back((temp >> 8 ) & 0x000000FF);
    decodedBytes.push_back((temp      ) & 0x000000FF);
  }
  return decodedBytes;
}
/* Counts the number of different bits between two byte vectors
 * Input:
 *	a,b: Strings to compare
 */
unsigned int countDiff(const std::vector<BYTE>& a, const std::vector<BYTE>& b)
{
  unsigned int buf=0;
	for(unsigned int i=0;i<a.size();i++)
		buf+=(std::bitset<8>(a[i]^b[i])).count();
	return buf;
}
NAN_METHOD(hashCompareCpp)
{
  if(info.Length()<3){
    Nan::ThrowTypeError("Wrong number of arguments");
    info.GetReturnValue().Set(-1);
    return;
  }
  if(!info[0]->IsNumber() || !info[1]->IsString() || !info[2]->IsArray()){
    Nan::ThrowTypeError("Wrong arguments");
    info.GetReturnValue().Set(-1);
    return;
  }

  unsigned int threshold = info[0]->Uint32Value();


  std::vector<BYTE> posted = base64Decode(*Nan::Utf8String(info[1]));

  Handle<Array> toTest = Handle<Array>::Cast(info[2]);
  for(unsigned int i=0; i<toTest->Length();i++) {	//Compare posted with other hashes
    String::Utf8Value param1(toTest->Get(i)->ToString()); //change from node string to c++ string
    std::string numHash = std::string(*param1);

    unsigned int numPos = numHash.find_first_of(':');
    std::vector<BYTE> tested = base64Decode(numHash.substr(numPos+1,numHash.length()));
    if(countDiff(posted,tested)<threshold){
	    info.GetReturnValue().Set(atoi(numHash.substr(0,numPos).c_str()));
	    return;
    }
  }
  info.GetReturnValue().Set(0);
}

NAN_MODULE_INIT(Init){
  Nan::Set(target, 
	   Nan::New<String>("hashCompareCpp").ToLocalChecked(),
	   Nan::New<FunctionTemplate>(hashCompareCpp)->GetFunction());
}
NODE_MODULE(compare,Init)
