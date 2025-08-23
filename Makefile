POMODORO_BINARY=pomodoroApp
UML_DIR := docs/uml
PUML := $(UML_DIR)/ClassDiagram.puml
PNG  := $(UML_DIR)/ClassDiagram.png
up:
	@echo "Starting Docker images.."
	docker-compose up -d
	@echo "Docker images started!"


up_build: build_pomodoro_service
	@echo "Stopping docker images (if running...)"
	docker-compose down
	@echo "Building (when required) and starting docker images..."
	docker-compose up --build -d
	@echo "Docker images built and started!"

build_pomodoro_service:
	@echo "Building pomodoro service..."
	env GOOS=linux CGO_ENABLED=0 go build -o ${POMODORO_BINARY} ./cmd/main.go
	@echo "Pomodoro service built!"

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


.PHONY: uml clean

uml: $(PNG)

$(PNG): $(PUML)
	@echo "Rendering PNG from $(PUML) -> $(PNG)"
	@plantuml -tpng $(PUML)

$(PUML):
	@echo "Generating PlantUML to $(PUML)"
	@mkdir -p $(UML_DIR)
	@goplantuml -recursive ./ > $(PUML)

clean:
	@rm -f $(PUML) $(PNG)
