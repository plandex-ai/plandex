FROM golang:1.23.3

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

# Build the application
RUN rm -f plandex-server && go build -o plandex-server .

# Set the port and expose it
ENV PORT=8099
EXPOSE 8099

# Command to run the executable
CMD ["./plandex-server"]
