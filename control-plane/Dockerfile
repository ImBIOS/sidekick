# Self-host image for the org-scoped scheduler (builds/sims/emus).
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY control-plane/ ./control-plane/
RUN CGO_ENABLED=0 go build -o /control-plane ./control-plane

FROM gcr.io/distroless/static-debian12
COPY --from=build /control-plane /control-plane
EXPOSE 8080
ENTRYPOINT ["/control-plane"]
