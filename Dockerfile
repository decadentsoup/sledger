FROM us-east4-docker.pkg.dev/shared-svcs-prod-artifacts-e7/shared-svcs-docker-images/golang:1.18 AS build

RUN apt update && apt install -y postgresql-client

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 go build -o sledger

FROM us-east4-docker.pkg.dev/shared-svcs-prod-artifacts-e7/shared-svcs-docker-images/golang:1.18
COPY --from=build /app/sledger /sledger
COPY entrypoint.sh /entrypoint.sh
WORKDIR /ledger
ENTRYPOINT ["/entrypoint.sh"]
