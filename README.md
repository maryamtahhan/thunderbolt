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
$ ./thunderbolt  -h
A utility to manage GPU Kernel runtime container images

Usage:
  thunderbolt [flags]

Flags:
  -c, --create         Create an OCI image
  -e, --extract        Extract an OCI image
  -h, --help           help for thunderbolt
  -i, --image string   OCI image name
```

## Triton Cache Image Container Specification

The Triton Cache Image specification defines how to bundle Triton Caches
as container images. A compatible Triton Cache image consists of cache
directory for a Triton Kernel.

There are two variants of the specification:

- [spec.md](./spec.md)
- [spec-compat.md](./spec-compat.md)