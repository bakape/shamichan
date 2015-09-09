#ifndef COMPARE_H_
#define COMPARE_H_

#include <vector>

#define TCHAR char    //Not unicode
#define TEXT(x) x     //Not unicode
#define DWORD long
#define BYTE unsigned char

unsigned int countDiff(const std::vector<BYTE>& a, const std::vector<BYTE>& b);
std::vector<BYTE> base64Decode(const std::basic_string<TCHAR>& input);
#endif //COMPARE_H_
