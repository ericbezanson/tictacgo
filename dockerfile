# Use an official Go image as the base
FROM golang:alpine

# Set the working directory to /app
WORKDIR /app

# Copy the go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the application code
COPY . .

RUN echo "Environment: ${REDIS_ADDRESS}"

# Build the application
RUN go build -o main ./cmd

# Expose the port
EXPOSE 8080

# Run the command to start the application
CMD ["./main"]
