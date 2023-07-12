FROM golang:1.20-alpine AS build
WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 go build -o sledger

FROM scratch
COPY --from=build /app/sledger /sledger
WORKDIR /ledger
ENTRYPOINT ["/sledger"]
