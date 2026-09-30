FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
RUN apk add --no-cache ca-certificates && adduser -D -u 1000 liki
RUN mkdir -p /var/lib/liki && chown liki:liki /var/lib/liki
COPY bin/liki-web /usr/local/bin/liki-web
COPY web/ /web/
LABEL org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_TIME}" \
      org.opencontainers.image.title="liki-web" \
      org.opencontainers.image.description="Liki product BFF and web application"
USER liki
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/liki-web"]
