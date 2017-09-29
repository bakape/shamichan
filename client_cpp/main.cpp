#include <emscripten.h>

int main() {
  EM_ASM("document.getElementById('threads').innerHTML = 'Hello World!'");
  return 0;
}
