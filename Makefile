CC = gcc
CC_FLAGS = -Wall -Wextra -pedantic -ggdb
CC_LIBS = -lm
OPTIMIZATION_FLAGS = 

ifdef MCA_OPTIMIZE
	OPTIMIZATION_FLAGS = -O3
endif

bin/mca: main.o lexer.o log.o ast.o io.o evaluator.o
	$(CC) $(CC_FLAGS) $(OPTIMIZATION_FLAGS) -o ./bin/mca $^ $(CC_LIBS)

bin/test: test.o lexer.o log.o ast.o io.o evaluator.o
	$(CC) $(CC_FLAGS) -o ./bin/test $^ $(CC_LIBS)

log.o: log.c log.h
	$(CC) $(CC_FLAGS) -c log.c $(CC_LIBS)

main.o: main.c lexer.h lexer.c log.h log.c
	$(CC) $(CC_FLAGS) -c main.c $(CC_LIBS)

lexer.o: lexer.c lexer.h log.h log.c
	$(CC) $(CC_FLAGS) -c lexer.c $(CC_LIBS)

ast.o: ast.c ast.h lexer.h lexer.c
	$(CC) $(CC_FLAGS) -c ast.c $(CC_LIBS)

io.o: io.c io.h
	$(CC) $(CC_FLAGS) -c io.c $(CC_LIBS)

evaluator.o: evaluator.c evaluator.h
	$(CC) $(CC_FLAGS) -c evaluator.c $(CC_LIBS)

test.o: test.c lexer.h lexer.c ast.h ast.c evaluator.h evaluator.c
	$(CC) $(CC_FLAGS) -c test.c $(CC_LIBS)

clean:
	rm -rf *.o bin/mca
