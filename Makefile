./bin/mca:
	cd src && go build -o ../bin/mca cmd/mca/main.go 

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
	./examples/loops.mca
	./examples/closures.mca
	./examples/help.mca
	cd ./examples/module && ./main.mca '$(shell echo -e '1, 2,      3, \n456    ')' && cd ../../

.PHONY: ./bin/mca exec_examples
