CC=gcc
CFLAGS=

loop: loop.c
	$(CC) -o loop loop.c $(CFLAGS)

.PHONY: clean

clean:
	find . -maxdepth 1 -type f ! -name '*.c' ! -name 'Makefile' -exec rm -f {} \;
