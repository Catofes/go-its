all :
	mkdir -p build
	go build -o build/client github.com/Catofes/go-its/application/client
	go build -o build/server github.com/Catofes/go-its/application/server
