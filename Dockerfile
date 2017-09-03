FROM scratch

COPY pkg/ca-certificates.crt /etc/ssl/certs/

COPY pkg/moira.yml /
COPY pkg/storage-schemas.conf /
COPY build/moira /

# relay
EXPOSE 2003 2003
# api
EXPOSE 8081 8081

ENTRYPOINT ["/moira"]
