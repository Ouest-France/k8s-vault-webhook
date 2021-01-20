FROM alpine AS certs

FROM scratch

# Import CA certificates
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy compiled static binary
COPY k8s-vault-webhook /k8s-vault-webhook

# Expose default port
EXPOSE 8443

ENTRYPOINT ["/k8s-vault-webhook"]