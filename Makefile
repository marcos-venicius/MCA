CC = gcc
CC_FLAGS = -Wall -Wextra -pedantic
CC_LIBS = -lm
OPTIMIZATION_FLAGS = 

ifdef MCA_OPTIMIZE
	CC_FLAGS += -O3
else
	CC_FLAGS += -ggdb
endif

bin/mca: main.o lexer.o ast.o io.o interpreter.o builtins/map.o ht.o arena.o
	$(CC) $(CC_FLAGS) -o ./bin/mca $^ $(CC_LIBS)

bin/test: test.o lexer.o ast.o io.o interpreter.o builtins/map.o ht.o arena.o
	$(CC) $(CC_FLAGS) -o ./bin/test $^ $(CC_LIBS)

all: bin/mca bin/test

exec_examples:
	./examples/empty.mca
	./examples/exiting-the-program.mca || :
	./examples/fib.mca
	./examples/fib2.mca
	./examples/hello.mca World
	./examples/is-leap-year-utc.mca || :
	./examples/logical-operators.mca
	./examples/math.mca
	./examples/maps.mca
	./examples/arrays.mca
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

io.o: io.c io.h
	$(CC) $(CC_FLAGS) -c io.c $(CC_LIBS)

ht.o: ht.c ht.h
	$(CC) $(CC_FLAGS) -c ht.c $(CC_LIBS)

arena.o: arena.c arena.h
	$(CC) $(CC_FLAGS) -c arena.c $(CC_LIBS)

builtins/map.o: ./builtins/map.c ./builtins/map.h
	$(CC) $(CC_FLAGS) -c ./builtins/map.c -o ./builtins/map.o $(CC_LIBS)

lexer.o: lexer.c lexer.h
	$(CC) $(CC_FLAGS) -c lexer.c $(CC_LIBS)

ast.o: ast.c ast.h lexer.h lexer.c colors.h constraints.h
	$(CC) $(CC_FLAGS) -c ast.c $(CC_LIBS)

interpreter.o: interpreter.c interpreter.h colors.h constraints.h
	$(CC) $(CC_FLAGS) -c interpreter.c $(CC_LIBS)

main.o: main.c lexer.h lexer.c ast.h ast.c interpreter.h interpreter.c ht.h ht.c builtins/map.h builtins/map.c
	$(CC) $(CC_FLAGS) -c main.c $(CC_LIBS)

test.o: test.c lexer.h lexer.c ast.h ast.c interpreter.h interpreter.c ht.h ht.c builtins/map.h builtins/map.c
	$(CC) $(CC_FLAGS) -c test.c $(CC_LIBS)

clean:
	rm -rf $(shell find . -type f -iname '*.o') bin/mca bin/test

.PHONY: all clean
