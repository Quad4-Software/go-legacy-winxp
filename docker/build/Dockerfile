# Build the go-legacy-winxp toolchain.
# docker build -f docker/build/Dockerfile -t go-legacy-winxp .
FROM golang:1.24.6 AS bootstrap

ENV CGO_ENABLED=0
WORKDIR /src
COPY . /src
WORKDIR /src/src
RUN ./make.bash

FROM debian:bookworm-slim
COPY --from=bootstrap /src /usr/local/go-legacy-winxp
ENV GOROOT=/usr/local/go-legacy-winxp
ENV GOPATH=/go
ENV PATH=/usr/local/go-legacy-winxp/bin:/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
WORKDIR /work
CMD ["go", "version"]
