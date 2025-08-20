FROM alpine:latest

RUN mkdir /app

COPY pomodoroApp /app

CMD ["/app/pomodoroApp"]