#ifndef APNGDETECTOR_H
#define APNGDETECTOR_H

#include <nan.h>
#include <string>
class apngDetector : public Nan::ObjectWrap
{
public:
  static NAN_MODULE_INIT(Init);
private:
  apngDetector();
  ~apngDetector();

  static NAN_METHOD(New);
  static NAN_METHOD(Detect);
  static Nan::Persistent<v8::Function> constructor;

  int checkChunk(const unsigned char* buffer,uint& offset);
  int cOffset;
  std::string cBytes;
};
#endif // APNGDETECTOR_H
