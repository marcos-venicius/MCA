./bin/mca:
	cd src && go build -o ../bin/mca cmd/mca/main.go 

.PHONY: ./bin/mca
