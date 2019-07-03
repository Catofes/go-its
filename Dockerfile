FROM golang AS builder
RUN go get -u github.com/golang/dep/cmd/dep
WORKDIR $GOPATH/src/github.com/Catofes/go-its
COPY . ./
RUN dep ensure --vendor-only
RUN CGO_ENABLED=0 make

FROM ubuntu:16.04
RUN apt update && apt install -y ca-certificates
RUN mkdir /usr/app
COPY --from=builder /go/src/github.com/Catofes/go-its/build/its ./
WORKDIR /usr/app
CMD ["/usr/app/its","-conf","/usr/app/its.json"]
