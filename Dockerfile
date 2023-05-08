FROM golang:1.20.1-alpine AS build_base

RUN apk add --no-cache git build-base

# Set the Current Working Directory inside the container
WORKDIR /tmp/billwise-server

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Unit tests
RUN CGO_ENABLED=1 go test -v

# Build the Go app
RUN go build -ldflags "-s -w" -o ./out/billwise-server .

# Start fresh from a smaller image
FROM alpine:3.9 
RUN apk add ca-certificates

WORKDIR /app

# Copy the Pre-built binary file from the previous stage. Observe we also copied the .env file
COPY --from=build_base /tmp/billwise-server/out/billwise-server .
COPY --from=build_base /tmp/billwise-server/.env .

RUN mkdir uploads

#RUN ./billwise-server permissions admin edit_tasks edit_activities edit_accounting edit_invoices

# Run the binary program produced by `go install`
CMD ["/app/billwise-server","serve"]