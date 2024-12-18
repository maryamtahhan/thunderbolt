# Thunderbolt

<img src="logo/thunderbolt.jpeg" alt="thunderbolt" width="20%" height="auto">

A GPU Kernel container image management utility inspired by
[WASM](https://github.com/solo-io/wasm/blob/master/spec/README.md).

## Build

```bash
$ go build
```

## Usage

```bash
$ ./thunderbolt  -h
A tool to manage GPU Kernel container images

Usage:
  thunderbolt [flags]

Flags:
  -c, --create         Create an OCI image
  -e, --extract        Extract an OCI image
  -h, --help           help for thunderbolt
  -i, --image string   OCI image name
```