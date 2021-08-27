#include <unistd.h>

int main(void) {
  for (;;) {
    write(STDOUT_FILENO, "Message\n", 8);
    sleep(3);
  }

  return 0;
}

