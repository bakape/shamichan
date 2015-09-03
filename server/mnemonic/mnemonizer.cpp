/*
 * Mnemonizer.cpp
 * Object to convert random ipv4 and ipv6 to a mnemonic.
 * The gist of the algorithm works like this:
 *      Checks if the ip is valid ipv4 or ipv6
 *      Adds a static salt to the string
 *      Hashes the ip and the salt with SHA1
 *      Splits the hash into 4 parts
 *      Gets one of the mnemonicStart and one of the mneomicends for each part
 *
 * To use it:
 *  First require it:
 * 	require('./path/mnemonics.node');
 *  then create an object with a salt like this:
 *      var mnem = new mnemonics.mnemonizer("This is an example salt[0._\Acd2*+ç_SAs]");
 *  The salt is reccommended to be SALTLENGTH long
 *  Then you can call Apply_mnemonic with any ip
 *      mnem.Apply_mnemonic("192.168.1.1");
 *  This will return
 *      The mnemonic if the argument passed is a valid ip
 *      NULL if the ip was invalid or there was some incorrect parameter, and an error message will be printed
 */
#include "mnemonizer.h"
#include <iostream>
#include <sstream> //String stream
#include <cstdlib> //strtol
#include <cmath> //floor
#include <openssl/sha.h>

using namespace v8;
const std::string MNEMONICSTARTS[] = {"", "k", "s", "t", "d", "n", "h", "b",
                                "p", "m", "f", "r", "g", "z", "l", "ch"};
const std::string MNEMONICENDS[] = {"a", "i", "u", "e", "o", "a", "i", "u",
                                "e", "o", "ya", "yi", "yu", "ye", "yo", "'"};
const char* const HEXS = "0123456789ABCDEF";
const int SALTLENGTH = 40; //feel free to change this if you know a better length


NAN_MODULE_INIT(mnemonizer::Init){
  Local<FunctionTemplate> tpl = Nan::New<FunctionTemplate>(New);
  tpl->SetClassName(Nan::New<String>("mnemonizer").ToLocalChecked());
  tpl->InstanceTemplate()->SetInternalFieldCount(1);

  Nan::SetPrototypeMethod(tpl,"Apply_mnemonic",Apply_mnemonic);

  constructor.Reset(tpl->GetFunction());
  Nan::Set(target,
	   Nan::New<String>("mnemonizer").ToLocalChecked(),
	   tpl->GetFunction());
}

NAN_METHOD(mnemonizer::New){
  if(info.IsConstructCall()){
    mnemonizer* obj;
    if(!info[0]->IsString()){
        std::cerr<<"mnemonizer: Warning, invalid Salt, no salt will we used (non secure)"<<std::endl;
        obj= new mnemonizer(std::string(""));
    }else{
        String::Utf8Value saltUTF(info[0]);
        obj = new mnemonizer(std::string(*saltUTF));
    }
    obj->Wrap(info.This());
    info.GetReturnValue().Set(info.This());
  }else {
    Local<Function> cons = Nan::New<Function>(constructor);
    info.GetReturnValue().Set(cons->NewInstance());
  }
}

Nan::Persistent<Function> mnemonizer::constructor;
mnemonizer::mnemonizer(std::string salt)
{
    if(salt.length()<SALTLENGTH)
        std::cerr<<"mnemonizer: Warning, salt should be larger, at least "<<SALTLENGTH<<" characters"<<std::endl;
    this->salt=salt;
}

mnemonizer::~mnemonizer()
{
}


bool mnemonizer::isIpv4(std::string ip){
    if(*ip.rbegin()=='.')
        return false;
    std::stringstream ipStream(ip);
    std::string part;
    int count =0;
    char *end;
    while(std::getline(ipStream,part,'.')){
        if(count>3 || part.empty())
            return false;
        int val = strtol(part.c_str(),&end,10);
        if(*end!= '\0' ||val<0 || val>255)
            return false;
        count++;
    }
    return true;
}

bool mnemonizer::isIpv6(std::string ip){
    std::string::reverse_iterator lastC=ip.rbegin();
    if(ip.length()<2 ||(lastC[0]==':' && lastC[1]!=':')) //ends with ::
        return false;

    bool gap = false;
    std::string::iterator firstC = ip.begin();
    if(firstC[0]==':' && firstC[1]==':'){ //starts with ::
        gap=true;
        ip.erase(0,2);
    }
    std::stringstream ipStream(ip);
    std::string part;
    int count =0;
    bool canIpv4=true;
    
    char *end;
    while(std::getline(ipStream,part,':')){
        if(part.empty()){
            if(gap) //If we already have a gap don't accept this one
                return false;
            gap = true;
        }
        int val = strtol(part.c_str(),&end,16);
        if(*end!= '\0' || val>0xFFFF)
            return false;
	
	//Check for special case (::ffff:{Ipv4}) example ::ffff:192.168.1.100
	if(canIpv4){
		if(val==0xFFFF &&(gap || count==5)){
			if(isIpv4(ipStream.str().substr(ipStream.tellg())))
				return true;
		}
		if(val!=0 ||count==5)
			canIpv4=false;
	}
        count++;
    }
    if(!gap && count!=8)
        return false;
    return true;
}

std::string mnemonizer::hashToMem(unsigned char* hash){
    std::string result;
    result.reserve(10);
    for(int i=0;i<4;i++){
        std::string part = ucharToHex(hash+(i*5),8);
        unsigned long int val = strtoul(part.c_str(),NULL,16);
        result.append(
                    MNEMONICSTARTS[(int)floor((val%256)/16)]+
                    MNEMONICENDS[val%16]
                );
    }
    return result;
}

std::string mnemonizer::ucharToHex(unsigned char* hashpart,int length_hex){
    std::string out;
    out.reserve(length_hex);
    for(int i=0;i <length_hex/2;i++)
    {
        unsigned char c = hashpart[i];
        out.push_back(HEXS[c>>4]);
        out.push_back(HEXS[c&15]);
    }
    return out;
}

NAN_METHOD(mnemonizer::Apply_mnemonic){
    mnemonizer* obj = ObjectWrap::Unwrap<mnemonizer>(info.Holder());

    if(!info[0]->IsString()){
        std::cerr<<"mnemonizer: Wrong argument passed to Apply_mnemonic(Should be an String)"<<std::endl;
	info.GetReturnValue().SetNull();
	return;
    }

    std::string ip=*Nan::Utf8String(info[0]->ToString());
    if(obj->isIpv4(ip) || obj->isIpv6(ip)){
        unsigned char out[SHA_DIGEST_LENGTH];
        ip.append(obj->salt);
        SHA1((unsigned char*)ip.c_str(),ip.length(),out);
        info.GetReturnValue().Set(Nan::New<String>(obj->hashToMem(out)).ToLocalChecked());
	return;
    }
    std::cerr<<"mnemonizer: Ip not valid: "<<ip<<std::endl;
    info.GetReturnValue().SetNull();
}