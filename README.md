# Thunderbolt

<img src="logo/thunderbolt.jpeg" alt="thunderbolt" width="20%" height="auto">

A GPU Kernel runtime container packaging utility inspired by
[WASM](https://github.com/solo-io/wasm/blob/master/spec/README.md).

## Build

```bash
$ go build
```

## Usage

```bash
$ ./thunderbolt -h
A GPU Kernel runtime container image management utility

Usage:
  thunderbolt [flags]

Flags:
  -c, --create         Create - TODO
  -e, --extract        Extract a Triton cache from an OCI image
  -h, --help           help for thunderbolt
  -i, --image string   OCI image name
```

> NOTE: The create option is a work in progress. For now
to create an OCI image containing a Triton cache directory
please follow the instructions in
[spec-compat.md](./spec-compat.md).

## Triton Cache Image Container Specification

The Triton Cache Image specification defines how to bundle Triton Caches
as container images. A compatible Triton Cache image consists of cache
directory for a Triton Kernel.

There are two variants of the specification:

- [spec.md](./spec.md)
- [spec-compat.md](./spec-compat.md)

## Example

To extract the Triton Cache for the
[01-vector-add.py](https://github.com/triton-lang/triton/blob/main/python/tutorials/01-vector-add.py)
tutorial from [Triton](https://github.com/triton-lang/triton), run the following:

```bash
./thunderbolt -e -i quay.io/mtahhan/triton-cache:01-vector-add-latest
Img fetched successfully!!!!!!!!
Img Digest: sha256:b6d7703261642df0bf95175a64a01548eb4baf265c5755c30ede0fea03cd5d97
Img Size: 525
bash-4.4#
```
This will extract the cache directory from the `quay.io/mtahhan/triton-cache:01-vector-add-latest`
container image and copy it to  `~/.triton/cache/`.

To Create an OCI image for a Triton Cache using docker run the following:

```bash
./_output/bin/linux_amd64/thunderbolt -c -i quay.io/mtahhan/01-vector-add-cache -d ./example/01-vector-add-cache
Dockerfile generated successfully at ./Dockerfile
--> FROM scratchlsxbp72edslewmsk7ad51zrh as 0
--> LABEL org.opencontainers.image.title=01-vector-add-cache
--> COPY ./example/01-vector-add-cache ./io.triton.cache
--> Committing changes to quay.io/mtahhan/01-vector-add-cache ...
--> Done
Docker image built successfully
OCI image pushed successfully.
```

To see the new image:

```bash
 docker images
REPOSITORY                                                                                TAG                   IMAGE ID       CREATED          SIZE
quay.io/mtahhan/01-vector-add-cache                                                       latest                32572653bbbd   5 minutes ago    0B
```

To inspect the image with Skopeo

```bash
skopeo inspect docker-daemon:quay.io/mtahhan/01-vector-add-cache:latest
{
    "Name": "quay.io/mtahhan/01-vector-add-cache",
    "Digest": "sha256:702c8489ea8bf2565f863d5a1bf46b53a55b100d075c9118072ff812a57ff8b2",
    "RepoTags": [],
    "Created": "2025-01-24T14:04:22.696839184Z",
    "DockerVersion": "27.1.1",
    "Labels": {
        "org.opencontainers.image.title": "01-vector-add-cache"
    },
    "Architecture": "amd64",
    "Os": "linux",
    "Layers": [
        "sha256:5f70bf18a086007016e948b04aed3b82103a36bea41755b6cddfaf10ace3c6ef"
    ],
    "LayersData": [
        {
            "MIMEType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
            "Digest": "sha256:5f70bf18a086007016e948b04aed3b82103a36bea41755b6cddfaf10ace3c6ef",
            "Size": 1024,
            "Annotations": null
        }
    ],
    "Env": [
        "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
    ]
}
```