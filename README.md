# Bazel Build Distributed Cache

[![Go Report Card](http://goreportcard.com/badge/mwitkow/bazel-distcache)](http://goreportcard.com/report/mwitkow/bazel-distcache)
[![GoDoc](http://img.shields.io/badge/GoDoc-Reference-blue.svg)](https://godoc.org/github.com/mwitkow/bazel-distcache)
[![Apache 2.0 License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A distributed build cache for [Bazel](https://bazel.build/) declarative build system.

## Why?!

Bazel is a great tool for having reproducible builds. However, if you run a large CI/CD pipeline using it, you end up
rebuilding many artifacts over and over across different agents (or containers).

Bazel > 0.5.3 has **alpha** support for the future Google Remote Execution API (protobuf [here](https://github.com/googleapis/googleapis/blob/master/google/devtools/remoteexecution/v1test/remote_execution.proto)*[]:

The goal of the project is to leverage it and build:
 * `localcache` - local daemon that `bazel` talks to, providing low latency-first caching layer
 * `distcache` - remote server that `localcache` talks to in case it encounters a miss

Because Bazel supports hermetic builds, this cache will be usable for *all* languages and build targets: Java,
C++, Go, Python, protobuf, Scala... you name it.

## Status

This is **pre-alpha**, basically a `localcache` proof of concept. It works, and `bazel` happily uses it as a cache.

## Usage:

#### `localcache`

To build:

```
go install github.com/mwitkow/bazel-distcache/cmd/localcache
```

To start:
```
bin/localcache --blobstore_ondisk_path=/tmp/localcache/blobstore --actionstore_ondisk_path=/tmp/localcache/actionstore
```
At this point an HTTP debug interface (including metrics) is running on http://localhost:10100. The default gRPC address
for bazel is `localhost:10101`. You can use it for example:
```
bazel --host_jvm_args=-Dbazel.DigestFunction=SHA1 build  --strategy=Javac=remote --strategy=Closure=remote --spawn_strategy=remote --remote_cache=localhost:10101 ...
```

## Hacking Tips

 * you can enable gRPC tracing on https://localhost:10100/debug/requests with `--grpc_tracing_enabled` for easier debugging
 * this uses the compiled out `google.golang.org/genproto/googleapis` Go protobufs




## License

`bazel-distcache` is released under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details.
