FROM golang:1.21-alpine AS build
WORKDIR /go/src/app
COPY go.mod go.sum main.go ./
COPY internal/ internal/
RUN go mod download
RUN go build -o /go/bin/app

FROM gcr.io/distroless/static-debian12
COPY --from=build /go/bin/app /
EXPOSE 53
CMD ["/app"]
