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
./_output/bin/linux_amd64/thunderbolt -e -i quay.io/mtahhan/triton-cache:01-vector-add-latest
Img fetched successfully!!!!!!!!
Img Digest: sha256:b6d7703261642df0bf95175a64a01548eb4baf265c5755c30ede0fea03cd5d97
Img Size: 525
bash-4.4#
```
This will extract the cache directory from the `quay.io/mtahhan/triton-cache:01-vector-add-latest`
container image and copy it to  `~/.triton/cache/`.

To Create an OCI image for a Triton Cache using docker run the following:

```bash
./_output/bin/linux_amd64/thunderbolt -c -i quay.io/mtahhan/01-vector-add-cache -d example/01-vector-add-cache
Current working directory: /home/mtahhan/thunderbolt
Dockerfile generated successfully at /home/mtahhan/thunderbolt/Dockerfile
{"stream":"Step 1/3 : FROM scratch"}
{"stream":"\n"}
{"stream":" ---\u003e \n"}
{"stream":"Step 2/3 : LABEL org.opencontainers.image.title=01-vector-add-cache"}
{"stream":"\n"}
{"stream":" ---\u003e Running in e984b66d8ba8\n"}
{"stream":" ---\u003e 252e1fe2dccf\n"}
{"stream":"Step 3/3 : COPY \"example/01-vector-add-cache/\" ./io.triton.cache/"}
{"stream":"\n"}
{"stream":" ---\u003e f14fef40f4cf\n"}
{"aux":{"ID":"sha256:f14fef40f4cf859010039b06cfcb4bfa3eedb3a259336679026f3784fd751ec2"}}
{"stream":"Successfully built f14fef40f4cf\n"}
{"stream":"Successfully tagged quay.io/mtahhan/01-vector-add-cache:latest\n"}
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
    "Digest": "sha256:3a6338dde949fd7158c2a7c54b17f866c3587e7c022b84ce443924f861335f2f",
    "RepoTags": [],
    "Created": "2025-01-27T10:45:28.225035278Z",
    "DockerVersion": "27.1.1",
    "Labels": {
        "org.opencontainers.image.title": "01-vector-add-cache"
    },
    "Architecture": "amd64",
    "Os": "linux",
    "Layers": [
        "sha256:6e28c76bcba8c174724befa53cbf7f36e7684609c7fefa13004bac257324f594"
    ],
    "LayersData": [
        {
            "MIMEType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
            "Digest": "sha256:6e28c76bcba8c174724befa53cbf7f36e7684609c7fefa13004bac257324f594",
            "Size": 82432,
            "Annotations": null
        }
    ],
    "Env": [
        "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
    ]
}
```