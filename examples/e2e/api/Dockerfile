FROM golang:1.25-alpine AS build

ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY examples/e2e/api ./examples/e2e/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /demo-api ./examples/e2e/api

FROM alpine:3.22

COPY --from=build /demo-api /demo-api
EXPOSE 8080
ENTRYPOINT ["/demo-api"]
