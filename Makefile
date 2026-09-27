.PHONY: run serve vet test ui-test build css words audio clean

run:
	go run . -dev -addr 127.0.0.1:8080

# Listen on all interfaces so other devices on the same Wi-Fi (e.g. an iPad)
# can reach it at http://<this-machine's-IP>:8080
serve:
	go run . -dev -addr 0.0.0.0:8080

vet:
	go vet ./...

test:
	go test ./...
	npm test

# Boot the binary and drive every page in headless Chromium (needs `npx playwright install chromium` once).
ui-test:
	node tests/ui/smoke.mjs

build: css
	go build -o skilldojo .

# Rebuild static/app.css after template/class changes (needs `npm install` once).
css:
	./node_modules/.bin/tailwindcss -i input.css -o static/app.css --minify

# Regenerate static/words.js from internal/curriculum/spelling.json.
words:
	node scripts/generate-words.js

# Rebuild the versioned spelling clips (macOS `say` + ffmpeg).
audio:
	npm run audio:spelling

clean:
	rm -f skilldojo
