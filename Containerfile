# -------------------------------------------------------------
FROM golang:alpine AS builder

COPY . /app
WORKDIR /app
RUN mkdir /out && GOFLAGS="-tags=no_fs_access,musl" go build -o /out/gossamer /app/cmd/gossamer

# -------------------------------------------------------------

FROM alpine

COPY --from=builder /out/gossamer /gossamer
COPY ./static/ /static/
COPY ./templates /templates/
COPY ./conf/ /conf/

LABEL org.opencontainer.image.url="https://github.com/eendcode/gossamer"
LABEL org.opencontainer.image.title="Gossamer"
LABEL org.opencontainer.description="Attacker-hindering Web Application Firewall"


USER 65535

CMD ["./gossamer"]