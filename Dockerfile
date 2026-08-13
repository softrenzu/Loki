FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go test ./... && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/rooomlog ./cmd/rooomlog
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/rooomlog /rooomlog
VOLUME ["/data"]
EXPOSE 3100
ENV ROOOMLOG_DATA=/data
ENTRYPOINT ["/rooomlog"]
