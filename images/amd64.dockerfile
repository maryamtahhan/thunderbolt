FROM public.ecr.aws/docker/library/golang:1.24 AS builder

COPY . /usr/src/thunderbolt
WORKDIR /usr/src/thunderbolt

ENV CGO_ENABLED=1
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgpgme-dev \
    libbtrfs-dev \
    build-essential \
    git \
    libc-dev \
    libffi-dev \
    linux-headers-amd64 \
 && rm -rf /var/lib/apt/lists/*

RUN make build

FROM public.ecr.aws/docker/library/debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    libgpgme11 \
    libbtrfs0 \
    libffi8 \
    libc6 \
 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/src/thunderbolt/_output/bin/linux_amd64/thunderbolt /thunderbolt
COPY images/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
