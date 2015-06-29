#ifndef MNEMONIZER_H
#define MNEMONIZER_H
#include <string>
#include <vector>
#include <nan.h>

class mnemonizer: public node::ObjectWrap
{
public:
  static void Init(v8::Handle<v8::Object> exports);
private:
    std::string salt;

    static NAN_METHOD(New);
    static NAN_METHOD(Apply_mnemonic);
    static v8::Persistent<v8::Function> constructor;

    mnemonizer(std::string salt);
    ~mnemonizer();
    std::string hashToMem(unsigned char* hash);
    bool isIpv4(std::string ip);
    bool isIpv6(std::string ip);
    std::string ucharToHex(unsigned char *hashpart, int length_hex);
};

#endif // MNEMONIZER_H
