CC = gcc
CC_FLAGS = -Wall -Wextra -pedantic -ggdb
CC_LIBS = -lm

mca: main.o lexer.o log.o ast.o
	$(CC) $(CC_FLAGS) -o mca $^ $(CC_LIBS)

log.o: log.c log.h
	$(CC) $(CC_FLAGS) -c log.c $(CC_LIBS)

main.o: main.c lexer.h lexer.c log.h log.c
	$(CC) $(CC_FLAGS) -c main.c $(CC_LIBS)

lexer.o: lexer.c lexer.h log.h log.c
	$(CC) $(CC_FLAGS) -c lexer.c $(CC_LIBS)

ast.o: ast.c ast.h lexer.h lexer.c
	$(CC) $(CC_FLAGS) -c ast.c $(CC_LIBS)

clean:
	rm -rf *.o mca
