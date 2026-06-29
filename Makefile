CC = gcc
CC_FLAGS = -Wall -Wextra -pedantic
CC_LIBS = -lm
OPTIMIZATION_FLAGS = 

ifdef MCA_OPTIMIZE
	OPTIMIZATION_FLAGS = -O3
else
	CC_FLAGS += -ggdb
endif

bin/mca: main.o lexer.o log.o ast.o io.o interpreter.o
	$(CC) $(CC_FLAGS) $(OPTIMIZATION_FLAGS) -o ./bin/mca $^ $(CC_LIBS)

bin/test: test.o lexer.o log.o ast.o io.o interpreter.o
	$(CC) $(CC_FLAGS) -o ./bin/test $^ $(CC_LIBS)

exec_examples:
	./examples/exiting-the-program.mca || :
	./examples/fib.mca
	./examples/fib2.mca
	./examples/hello.mca World
	./examples/is-leap-year-utc.mca || :
	./examples/logical-operators.mca
	./examples/math.mca
	./examples/pascals-triangle.mca 5
	./examples/play.mca
	./examples/sleep.mca
	./examples/stopwatch.mca 2
	./examples/today.mca
	./examples/touring-complete.mca
	./examples/triangle-angle.mca 3 4 5
	./examples/type-casting.mca
	./examples/type-inspect.mca
	./examples/unit.mca
	./examples/user-defined-functions.mca

all: bin/mca bin/test

log.o: log.c log.h
	$(CC) $(CC_FLAGS) -c log.c $(CC_LIBS)

main.o: main.c lexer.h lexer.c log.h log.c ht.h builtins/map.h
	$(CC) $(CC_FLAGS) -c main.c $(CC_LIBS)

lexer.o: lexer.c lexer.h log.h log.c
	$(CC) $(CC_FLAGS) -c lexer.c $(CC_LIBS)

ast.o: ast.c ast.h lexer.h lexer.c
	$(CC) $(CC_FLAGS) -c ast.c $(CC_LIBS)

io.o: io.c io.h
	$(CC) $(CC_FLAGS) -c io.c $(CC_LIBS)

interpreter.o: interpreter.c interpreter.h
	$(CC) $(CC_FLAGS) -c interpreter.c $(CC_LIBS)

test.o: test.c lexer.h lexer.c ast.h ast.c interpreter.h interpreter.c
	$(CC) $(CC_FLAGS) -c test.c $(CC_LIBS)

clean:
	rm -rf *.o bin/mca bin/test

.PHONY: all clean
