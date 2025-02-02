CC = gcc
CC_FLAGS = -Wall -Wextra -pedantic -ggdb

mca: main.o lexer.o
	$(CC) $(CC_FLAGS) -o mca $<

main.o: main.c
	$(CC) $(CC_FLAGS) -c main.c

lexer.o: lexer.c lexer.h
	$(CC) $(CC_FLAGS) -c lexer.c

clean:
	rm -rf *.o mca
