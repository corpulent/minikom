FROM golang:1.16-alpine
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
COPY *.go ./
RUN go build -o /minikom

HEALTHCHECK --interval=5m --timeout=3s \
  CMD curl -f http://localhost:8080/health || exit 1
CMD [ "/minikom" ]