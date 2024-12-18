# Thunderbolt

<img src="logo/thunderbolt.jpeg" alt="thunderbolt" width="20%" height="auto">

A GPU Kernel runtime container image management utility inspired by
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