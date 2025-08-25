POMODORO_BINARY=pomodoroApp

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
	cd ../pomodoro-service && env GOOS=linux CGO_ENABLED=0 go build -o ${POMODORO_BINARY} ./cmd/api
	@echo "Pomodoro service built!"
