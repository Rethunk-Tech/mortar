# Site and packages are copied in by the deploy step; this image is the binary only.
FROM golang:1.27.4-bookworm AS build
WORKDIR /src
COPY . .
ENV CGO_ENABLED=0
RUN go build -o /sharehost ./cmd/sharehost

FROM gcr.io/distroless/static-debian12
COPY --from=build /sharehost /sharehost
USER nonroot
ENTRYPOINT ["/sharehost"]
