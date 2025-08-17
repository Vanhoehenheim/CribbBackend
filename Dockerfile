    # syntax=docker/dockerfile:1

    FROM golang:1.23-alpine AS build-stage
    WORKDIR /app
    COPY go.mod go.sum ./
    RUN go mod download
    COPY . .
    RUN CGO_ENABLED=0 GOOS=linux go build -o /cribb-backend-docker

    # Run the tests in the container
    FROM build-stage AS run-test-stage
    RUN go test -v ./...

    # Deploy the application binary into a lean image
    FROM gcr.io/distroless/static-debian12 AS build-release-stage

    WORKDIR /

    COPY --from=build-stage /cribb-backend-docker /cribb-backend-docker

    EXPOSE 8080
    USER nonroot:nonroot
    ENTRYPOINT ["/cribb-backend-docker"]