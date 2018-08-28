FROM alpine:latest as build
RUN  apk add --no-cache ca-certificates

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY server /stub-server
ENV PORT 8080
EXPOSE 8080
ENTRYPOINT [stub-server]
