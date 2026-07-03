FROM alpine:3.21 AS certs
RUN apk add --no-cache ca-certificates
RUN adduser -D -u 1000 liki

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=certs /etc/passwd /etc/passwd
COPY bin/liki-web /usr/local/bin/liki-web
USER liki
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/liki-web", "--addr", ":8081"]
