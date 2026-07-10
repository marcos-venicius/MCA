CC = gcc
CC_FLAGS = -Wall -Wextra -pedantic
CC_LIBS = -lm
OPTIMIZATION_FLAGS = 

ifdef MCA_OPTIMIZE
	CC_FLAGS += -O3
else
	CC_FLAGS += -ggdb
endif

SRCS = $(shell find . -type f -name '*.c')
COMMON_SRCS = $(filter-out ./main.c ./test.c, $(SRCS))
COMMON_OBJS = $(COMMON_SRCS:.c=.o)

../bin/mca: main.o $(COMMON_OBJS)
	$(CC) $(CC_FLAGS) -o ../bin/mca $^ $(CC_LIBS)

../bin/test: test.o $(COMMON_OBJS)
	$(CC) $(CC_FLAGS) -o ../bin/test $^ $(CC_LIBS)

all: ../bin/mca ../bin/test

# Generic rule for compiling C files to object files
# -MMD -MP automatically generates header dependency files (.d)
%.o: %.c
	$(CC) $(CC_FLAGS) -MMD -MP -c $< -o $@ $(CC_LIBS)

# Include generated dependency files
-include $(SRCS:.c=.d)

exec_examples:
	../examples/empty.mca
	../examples/exiting-the-program.mca || :
	../examples/fib.mca
	../examples/fib2.mca
	../examples/hello.mca World
	../examples/is-leap-year-utc.mca || :
	../examples/logical-operators.mca
	../examples/math.mca
	../examples/maps.mca
	../examples/arrays.mca
	../examples/pascals-triangle.mca 5
	../examples/play.mca
	../examples/sleep.mca
	../examples/stopwatch.mca 2
	../examples/today.mca
	../examples/touring-complete.mca
	../examples/triangle-angle.mca 3 4 5
	../examples/type-casting.mca
	../examples/type-inspect.mca
	../examples/unit.mca
	../examples/user-defined-functions.mca
	../examples/loops.mca
	cd ../examples/module && ./main.mca '$(shell echo -e '1, 2,      3, \n456    ')' && cd ../../c

clean:
	rm -rf $(shell find . -type f -name '*.o') $(shell find . -type f -name '*.d') ../bin/mca ../bin/test

.PHONY: all clean
