FROM alpine:3.21
RUN apk add --no-cache ca-certificates python3 && adduser -D -u 1000 liki
RUN mkdir -p /var/lib/liki && chown liki:liki /var/lib/liki
COPY bin/liki-web /usr/local/bin/liki-web
COPY web/ /web/
ENV LIKI_SKILLS_DIR=/web/skills
USER liki
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/liki-web", "--addr", ":8081"]
