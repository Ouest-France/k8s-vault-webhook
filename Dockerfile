#####
# Build stage
#####
FROM golang:1.13 AS builder

# Create source path and cd
RUN mkdir -p src/github.com/Ouest-France/k8s-vault-webhook

WORKDIR /go/src/github.com/Ouest-France/k8s-vault-webhook

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 go build -ldflags="-extldflags '-static' -w -s" -o k8s-vault-webhook

#####
# Run stage
#####
FROM scratch

# Copy compiled static binary
COPY --from=builder /go/src/github.com/Ouest-France/k8s-vault-webhook/k8s-vault-webhook /k8s-vault-webhook

# Import CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Expose default port
EXPOSE 8443

CMD ["/k8s-vault-webhook"]