.PHONY: build build-server build-client test lint migrate run dev clean

build: build-server build-client

build-server:
	go build -o bin/tenantzero-server ./server

build-client:
	go build -o bin/tenantzero ./client

test:
	go test ./... -v -timeout 60s

test-store:
	TEST_DB_URL="postgres://tenantzero:tenantzero@localhost:5432/tenantzero?sslmode=disable" \
	go test ./server/store/... -v

lint:
	go vet ./...

run: build-server
	DATABASE_URL="postgres://tenantzero:tenantzero@localhost:5432/tenantzero?sslmode=disable" \
	JWT_SECRET="dev-secret" \
	LABS_DIR="./labs" \
	./bin/tenantzero-server -migrations ./server/store/migrations

dev:
	docker compose up --build

clean:
	rm -rf bin/