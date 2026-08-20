FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/uploy .

FROM alpine:3.21

ENV RAILPACK_VERSION=0.37.0
ARG TARGETARCH

# Railpack and git are needed on the API host for source analysis. The pinned
# version must match the BuildKit frontend used by the source deploy pipeline.
RUN apk add --no-cache ca-certificates curl git \
 && case "${TARGETARCH}" in \
      amd64) railpack_target="x86_64-unknown-linux-musl" ;; \
      arm64) railpack_target="arm64-unknown-linux-musl" ;; \
      *) echo "unsupported architecture: ${TARGETARCH}" >&2; exit 1 ;; \
    esac \
 && curl -fsSL "https://github.com/railwayapp/railpack/releases/download/v${RAILPACK_VERSION}/railpack-v${RAILPACK_VERSION}-${railpack_target}.tar.gz" \
      | tar -xzf - -C /usr/local/bin railpack \
 && railpack --version | grep -F "railpack version ${RAILPACK_VERSION}"

RUN addgroup -S uploy && adduser -S -G uploy uploy

COPY --from=build /out/uploy /usr/local/bin/uploy

USER uploy
EXPOSE 8080

ENTRYPOINT ["uploy"]
