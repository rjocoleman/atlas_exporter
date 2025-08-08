# Multi-stage to include system CA certificates for TLS verification
# Stage 1: get CA bundle
FROM alpine:3.20 AS certs
RUN apk --no-cache add ca-certificates

# Final image: minimal scratch with binary + CA certs
FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY atlas_exporter /
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt
ENTRYPOINT ["/atlas_exporter"]
