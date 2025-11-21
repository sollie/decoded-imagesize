FROM alpine:edge AS builder

RUN apk add --no-cache \
    go \
    git \
    build-base \
    pkgconfig \
    libheif-dev \
    libwebp-dev \
    libde265-dev \
    x265-dev \
    aom-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o decoded-imagesize .

FROM alpine:edge

RUN apk add --no-cache \
    libheif \
    libwebp \
    libde265 \
    x265-libs \
    aom-libs \
    ca-certificates

COPY --from=builder /build/decoded-imagesize /usr/local/bin/decoded-imagesize

RUN chmod +x /usr/local/bin/decoded-imagesize

WORKDIR /data

ENTRYPOINT ["/usr/local/bin/decoded-imagesize"]
CMD ["--help"]
