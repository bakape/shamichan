#ifndef APNGDETECTOR_H
#define APNGDETECTOR_H

#include <nan.h>
#include <string>
class apngDetector : public node::ObjectWrap
{
public:
  static void Init(v8::Handle<v8::Object> exports);
private:
  apngDetector();
  ~apngDetector();

  static NAN_METHOD(New);
  static NAN_METHOD(Detect);
  static v8::Persistent<v8::Function> constructor;

  int checkChunk(const unsigned char* buffer,uint& offset);
  int cOffset;
  std::string cBytes;
};
#endif // APNGDETECTOR_H
