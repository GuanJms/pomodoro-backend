POMODORO_BINARY=pomodoroApp
UML_DIR := docs/uml
PUML := $(UML_DIR)/ClassDiagram.puml
PNG  := $(UML_DIR)/ClassDiagram.png
up:
	@echo "Starting Docker images.."
	docker-compose up -d
	@echo "Docker images started!"


up_build: build_pomodoro_service_arm64
	@echo "Stopping docker images (if running...)"
	docker-compose down
	@echo "Building (when required) and starting docker images..."
	docker-compose up --build -d
	@echo "Docker images built and started!"

build_pomodoro_service_arm64:
	@echo "Building pomodoro service..."
	env GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o ${POMODORO_BINARY} ./cmd
	@echo "Pomodoro service built!"

build_pomodoro_service_amd64:
	@echo "Building pomodoro service..."
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ${POMODORO_BINARY} ./cmd
	@echo "Pomodoro service built!"

down:
	@echo "Stopping docker images (if running...)"
	docker-compose down
	@echo "Docker images stopped!"

test:
	@echo "Running unit tests..."
	go test -v ./internal/clock/...

test_integration:
	@echo "Running integration tests..."
	go test -v ./test/...

test_all:
	@echo "Running all tests..."
	go test -v ./internal/clock/... ./test/...

test_coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./internal/clock/... ./test/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test_benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./internal/clock/... ./test/...

clean:
	@echo "Cleaning up..."
	rm -f ${POMODORO_BINARY}
	rm -f coverage.out coverage.html
	@echo "Cleanup complete!"


.PHONY: uml clean_uml

uml: $(PNG)

$(PNG): $(PUML)
	@echo "Rendering PNG from $(PUML) -> $(PNG)"
	@plantuml -tpng $(PUML)

$(PUML):
	@echo "Generating PlantUML to $(PUML)"
	@mkdir -p $(UML_DIR)
	@goplantuml -recursive ./ > $(PUML)

clean_uml:
	@rm -f $(PUML) $(PNG)


generate_sqlc:
	@echo "Generate pomodoro-service sqlc file"
	sqlc generate
	@echo "Done!"

# Database management targets

pomodoro_up: build_pomodoro_service_arm64
	@echo "Starting pomodoro service..."
	docker-compose up --build -d pomodoro-service
	@echo "Pomodoro service started!"

pomodoro_down: 
	@echo "Stopping pomodoro service..."
	docker-compose down pomodoro-service
	@echo "Pomodoro service stopped!"


db_up:
	@echo "Starting database services..."
	docker-compose up -d postgres redis
	@echo "Database services started!"

db_down:
	@echo "Stopping database services..."
	docker-compose down
	@echo "Database services stopped!"

db_reset:
	@echo "Resetting database (WARNING: This will delete all data)..."
	docker-compose down
	@echo "Removing database data..."
	sudo rm -rf db-data/
	@echo "Starting fresh database..."
	docker-compose up -d postgres redis
	@echo "Database reset complete!"

db_logs:
	@echo "Showing database logs..."
	docker-compose logs -f postgres

db_connect:
	@echo "Connecting to database..."
	docker-compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

db_migrate:
	@echo "Running database migrations..."
	docker-compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -f /docker-entrypoint-initdb.d/schema.sql

build_docker_image_push_arm64: build_pomodoro_service_arm64
	@echo "Building docker image..."
	docker buildx build --platform linux/arm64 -t jamesguan777/pomodoro-backend:latest .
	@echo "Docker image built!"
	@echo "Pushing docker image to docker hub..."
	docker push jamesguan777/pomodoro-backend:latest
	@echo "Docker image pushed to docker hub!"

build_docker_image_push_amd64: build_pomodoro_service_amd64
	@echo "Building docker image..."
	docker buildx build --platform linux/amd64 -t jamesguan777/pomodoro-backend_amd64:latest .
	@echo "Docker image built!"
	@echo "Pushing docker image to docker hub..."
	docker push jamesguan777/pomodoro-backend_amd64:latest
	@echo "Docker image pushed to docker hub!"
