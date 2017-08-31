FROM scratch
ADD build/ca-certificates.crt /etc/ssl/certs/
ADD pkg/moira.yml /
ADD build/moira /

# relay
EXPOSE 2003 2003
# api
EXPOSE 8081 8081

CMD ["/moira"]
