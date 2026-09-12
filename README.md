# kinetiq

A web-assembly extensible, composable, kafka processor that supports hot-reloading.

# Quickstart

Brings up Kafka, kinetiq, and a producer that emits a message every 300ms, so you can
watch records flow through a WASM module without writing any client code.

**Prerequisites:** Go 1.24+, Docker, and a running Docker daemon.

```bash
# 1. Build the example WASM module.
#    Required: the .wasm artifact is gitignored, and compose bind-mounts it.
#    Skipping this leaves Docker mounting a directory where a file should be.
make build-test-module

# 2. Start everything.
docker compose --profile all up
```

Once it is up:

| What | Where |
| --- | --- |
| Messages on both topics | Kafka UI at [localhost:10015](http://localhost:10015) |
| Traces | Jaeger at [localhost:16686](http://localhost:16686) |
| Metrics and logs | Grafana at [localhost:41000](http://localhost:41000) |
| kinetiq health | `curl localhost:8080/health` |

The example module echoes each message and logs it, so `kinetiq-test-topic-out` should
mirror `kinetiq-test-topic`.

Use `--profile minimal` instead of `--profile all` for just Kafka, kinetiq, and the
producer, with no observability stack.

## Trying hot reload

Hot reload is easiest to see when kinetiq runs on the host rather than in a container,
because it watches the module file directly.

```bash
# Terminal 1 - Kafka only
make start-kafka && make create-topics

# Terminal 2 - kinetiq on the host
make run-test-module-local

# Terminal 3 - edit examples/test_module.go, then rebuild in place
make hotswap-module-local
```

The rebuild writes to the file kinetiq is watching, which triggers a reload. Watch
terminal 2 for the new module taking over without a restart.

To load modules from S3 instead of local disk, set `S3_INTEGRATION_ENABLED=true` along
with `S3_INTEGRATION_BUCKET` and `S3_INTEGRATION_CHANGE_QUEUE`, then publish with
`make hotswap-s3`. kinetiq downloads the object on startup and again whenever the change
queue reports a new version.

# Overview

Developing Kafka applications in a polyglot microservice architecture can be time consuming.  You spend time building,
deploying, redeploying, and application startup or rollout.  Further, you wind up writing a lot of boilerplate when creating
producers & consumers, processing topics, populating caches, performing windowed aggregations, and wiring up various
sources and sinks outside of kafka.  The boilerplate burden is multiplicative if your environment supports development in
multiple languages.  Further, there are some well-known integration patterns that can address a large
variety of business problems (eg. routing, replicating, merging, aggregating, filtering, hydrating, projecting, etc.).
While libraries can solve a lot of the issues with code duplication and boilerplate, they don't necessarily solve the
the slow feedback loop problem -- because ultimately you still need to build and deploy an application.  Kinetiq aims to
solve the problem of message processing boilerplate and long feedback loops together so you can iterate quickly on getting the
messages you want with the data you need *now*, not an hour from now.

# Concept & Technology

The original concept for Kinetiq was to leverage a combination of a few technologies for their main benefits:
* Go - a highly performant and fast-building systems language to serve as the basis for the server
* WebAssembly - a performant and portable execution binary format with good language support
* Protobuf - a compact, and type-safe serialization format that supports declarative API contracts across a wide variety of languages

The initial use-case was being able to read a kafka topic, perform some operation on the data defined by a web assembly
module, and write the result to an output topic.  Further, the web assembly module artifact could be monitored for
changes via filesystem notifications or cloud provider change events and dynamically reloaded when necessary to
facilitate a fast deployment simply by recompiling locally or publishing a new version of the WASM module to your cloud
provider.

Evaluating that simple use-case, it's easy to see how this same concept and its benefits could apply more widely to
different sources/sinks, data formats, and messaging providers.


# Current Limitations

This project is very much a work-in-progress currently, but you can track the progress of a viable release on the
[milestones page](https://github.com/kjkondratuk/kinetiq/milestones)!

1. Ingest is not paused during a module reload. `Source.Enable`/`Disable` exist for this but are not yet wired up, so a message already in flight can fail against a runtime that is being swapped. A failed *fetch* is safe -- the running module is left in place -- but a failed *load* leaves the loader without a usable module until restart
2. Only works currently with Kafka inputs and outputs
3. Requires web assembly modules to be loadable by [wazero](https://wazero.io/) runtime
4. Only supports module change detection for S3 and local files currently

Please check out [issues](https://github.com/kjkondratuk/kinetiq/issues) to see what we're working on!


# Plugins

This project utilizes web assembly modules to support dynamic reloading of the application logic, and uses protocol buffers
to communicate data between the host and plugin.  Tools supporting this interaction are:
* [Buf CLI / Buf Schema Registry](https://buf.build/) - client generation & build tooling
* [wazero](https://wazero.io/) - web assembly execution environment
* [knqyf263/go-plugin](https://github.com/knqyf263/go-plugin) - web assembly plugin implementation

## Schema

The plugin schema and compiled SDKs are available at [buf.build (BSR)](https://buf.build/kjkondratuk/kinetiq)!

# Docker Images

Docker images are available from [Docker Hub](https://hub.docker.com/r/paintface07/kinetiq/tags) and are available in:
* Alpine - `linux/amd64`
* Alpine - `linux/arm64`

# Documentation

Further documentation on using kinetiq is available in [the wiki](https://github.com/kjkondratuk/kinetiq/wiki)