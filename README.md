# Bazel Build Distributed Cache

[![Go Report Card](http://goreportcard.com/badge/mwitkow/bazel-distcache)](http://goreportcard.com/report/mwitkow/bazel-distcache)
[![GoDoc](http://img.shields.io/badge/GoDoc-Reference-blue.svg)](https://godoc.org/github.com/mwitkow/bazel-distcache)
[![Apache 2.0 License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A distributed build cache for [Bazel](https://bazel.build/) declarative build system.

## Why?!

Bazel is a great tool for having reproducible builds. However, if you run a large CI/CD pipeline using it, you end up
rebuilding many artifacts over and over across different agents (or containers).

Bazel has **experimental** support for gRPC-based [remote workers/caches](https://github.com/bazelbuild/bazel/tree/1575652972d80f224fb3f7398eef3439e4f5a5dd/src/main/java/com/google/devtools/build/lib/remote).

The goal of the project is to leverage it and build:
 * `localcache` - local daemon that `bazel` talks to, providing low latency-first caching layer
 * `distcache` - remote server that `localcache` talks to in case it encounters a miss

Because Bazel supports hermetic builds, this cache will be usable for *all* languages and build targets: Java,
C++, Go, Python, protobuf, Scala... you name it.

## Status

This is **pre-alpha**, basically a `localcache` proof of concept. It works, and `bazel` happily uses it as a cache.

## Usage: `localcache`

To build:

```
go install github.com/mwitkow/bazel-distcache/cmd/localcache
```

To start:
```
bin/localcache --blobstore_ondisk_path=/tmp/localcache-blobstore
```
At this point an HTTP debug interface (including metrics) is running on http://localhost:10100. The default gRPC address
for bazel is `localhost:10101`. You can use it for example:
```
bazel --host_jvm_args=-Dbazel.DigestFunction=SHA1 build  --spawn_strategy=remote --remote_cache=localhost:10101 ...
```

Some enterprises have fairly restrictive networking environments. They typically operate [HTTP forward proxies](https://en.wikipedia.org/wiki/Proxy_server) that require user authentication. These proxies usually allow  HTTPS (TCP to `:443`) to pass through the proxy using the [`CONNECT`](https://tools.ietf.org/html/rfc2616#section-9.9) method. The `CONNECT` method is basically a HTTP-negotiated "end-to-end" TCP stream... which is exactly what [`net.Conn`](https://golang.org/pkg/net/#Conn) is :)

## Hacking Tips

 * you can enable gRPC tracing on https://localhost:10100/debug/requests with `--grpc_tracing_enabled` for easier debugging
 *
 * [`remote_protocol.proto`](https://github.com/bazelbuild/bazel/blob/master/src/main/protobuf/remote_protocol.proto) contains the protocol
 * [`GrpcActionCache.java`](https://github.com/bazelbuild/bazel/blob/master/src/main/java/com/google/devtools/build/lib/remote/GrpcActionCache.java) is the Bazel-side of the protocol, with some interesting assumptions
 * [`RemoteOptions.java`](https://github.com/bazelbuild/bazel/blob/master/src/main/java/com/google/devtools/build/lib/remote/RemoteOptions.java) contains the CLI parameters for caching in Bazel
 * if you want to rebuild protos, use `proto/protogen.sh`



## License

`bazel-distcache` is released under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details.
