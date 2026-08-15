.PHONY: run serve vet test build css clean

run:
	go run . -addr 127.0.0.1:8080

# Listen on all interfaces so other devices on the same Wi-Fi (e.g. an iPad)
# can reach it at http://<this-machine's-IP>:8080
serve:
	go run . -addr 0.0.0.0:8080

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
