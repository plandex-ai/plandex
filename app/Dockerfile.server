FROM --platform=linux/amd64 golang:1.21.3

# Update and install necessary packages including build tools for Tree-sitter
RUN apt-get update && \
  apt-get install -y git gcc g++ make

WORKDIR /app

# Copy go.mod and go.sum for shared and server, and install dependencies
COPY ./shared/go.mod ./shared/go.sum ./shared/
RUN cd shared && go mod download

COPY ./server/go.mod ./server/go.sum ./server/
RUN cd server && go mod download

# Copy the actual source code
COPY ./server ./server
COPY ./shared ./shared
COPY ./scripts /scripts

# Set working directory to server
WORKDIR /app/server

# Set environment variables for Go
ENV GOOS=linux
ENV GOARCH=amd64

# Build the application
RUN rm -f plandex-server && go build -o plandex-server .

# Set the port and expose it
ENV PORT=8080
EXPOSE 8080

# Command to run the executable
CMD ["./plandex-server"]
