FROM golang:1.23.3

# Update and install necessary packages including build tools for Tree-sitter
RUN apt-get update && \
  apt-get install -y git gcc g++ make

# Install Python and create a virtual environment for litellm passthrough
RUN apt-get install -y python3 python3-venv && \
    python3 -m venv /opt/venv
    
# Activate the virtual environment for all following RUN commands
ENV PATH="/opt/venv/bin:$PATH"

# Now install litellm/prisma in the virtual environment
RUN pip install --no-cache-dir "litellm==1.61.1" "fastapi==0.115.12" "uvicorn==0.34.1"

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
