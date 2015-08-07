#ifndef MNEMONIZER_H
#define MNEMONIZER_H
#include <string>
#include <vector>
#include <nan.h>

class mnemonizer: public Nan::ObjectWrap
{
public:
    static NAN_MODULE_INIT(Init);
private:
    std::string salt;

    static NAN_METHOD(New);
    static NAN_METHOD(Apply_mnemonic);
    static Nan::Persistent<v8::Function> constructor;

    mnemonizer(std::string salt);
    ~mnemonizer();
    std::string hashToMem(unsigned char* hash);
    bool isIpv4(std::string ip);
    bool isIpv6(std::string ip);
    std::string ucharToHex(unsigned char *hashpart, int length_hex);
};

#endif // MNEMONIZER_H
