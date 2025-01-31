package constants

import "os"

const (
	DockerCacheDirPrefix  = "docker-cache-dir-"
	BuildahCacheDirPrefix = "buildah-cache-dir-"
	PodmanCacheDirPrefix  = "podman-cache-dir-"
	TritonCacheDirName    = "io.triton.cache/"
)

var (
	TritonCacheDir = os.Getenv("HOME") + "/.triton/cache"
)
