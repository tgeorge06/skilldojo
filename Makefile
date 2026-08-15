.PHONY: run vet test build css clean

run:
	go run . -addr 127.0.0.1:8080

vet:
	go vet ./...

test:
	go test ./...

build: css
	go build -o skilldojo .

# Rebuild static/app.css after template/class changes (needs `npm install` once).
css:
	./node_modules/.bin/tailwindcss -i input.css -o static/app.css --minify

clean:
	rm -f skilldojo
