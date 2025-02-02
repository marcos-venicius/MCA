CC = gcc
CC_FLAGS = -Wall -Wextra -pedantic -ggdb

mca: main.o lexer.o log.o
	$(CC) $(CC_FLAGS) -o mca $^

log.o: log.c log.h
	$(CC) $(CC_FLAGS) -c log.c

main.o: main.c lexer.h lexer.c log.h log.c
	$(CC) $(CC_FLAGS) -c main.c

lexer.o: lexer.c lexer.h log.h log.c
	$(CC) $(CC_FLAGS) -c lexer.c

clean:
	rm -rf *.o mca
