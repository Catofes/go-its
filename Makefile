all :
	mkdir -p build
	env CGO_ENABLED=0 go build -o build/client github.com/Catofes/go-its/application/client
	env CGO_ENABLED=0 go build -o build/server github.com/Catofes/go-its/application/server
