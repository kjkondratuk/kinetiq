# kinetiq

A web-assembly extensible, composable, kafka processor that supports hot-reloading.

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

The initial use-case was being able to read a kafka topic (specifically on AWS MSK), perform some operation on
the data defined by a web assembly module, and write the result to an output topic.  Further, the web assembly module
artifact could be monitored for changes via filesystem notifications or cloud provider change events and dynamically
reloaded when necessary to facilitate a fast deployment simply by recompiling locally or publishing a new version of the WASM
module to your cloud provider.

Evaluating that simple use-case, it's easy to see how this same concept and its benefits could apply more widely to
different sources/sinks, data formats, and messaging providers.


# Current Limitations

This project is very much a work-in-progress currently, but you can track the progress of a viable release on the
[milestones page](https://github.com/kjkondratuk/kinetiq/milestones)!

1. Message processing will be briefly delayed during module reloading to avoid data loss
2. Only works currently with Kafka inputs and outputs
3. Only supports PLAINTEXT communication for Kafka at the moment
4. Doesn't support consumer groups yet, so it processes the whole topic on start
5. Requires web assembly modules to be loadable by [wazero](https://wazero.io/) runtime
6. Only supports module change detection for S3 and local files currently
7. No deployable docker image published to Docker Hub yet
8. Schema and compiled SDKs are not published automatically

Please check out [issues](https://github.com/kjkondratuk/kinetiq/issues) to see what we're working on!


# Plugins

This project utilizes web assembly modules to support dynamic reloading of the application logic, and uses protocol buffers
to communicate data between the host and plugin.  Tools supporting this interaction are:
* [Buf CLI / Buf Schema Registry](https://buf.build/) - client generation & build tooling
* [wazero](https://wazero.io/) - web assembly execution environment
* [knqyf263/go-plugin](https://github.com/knqyf263/go-plugin) - web assembly plugin implementation

## Schema

The plugin schema and compiled SDKs are available at [buf.build (BSR)](https://buf.build/kjkondratuk/kinetiq)!

# Documentation

Further documentation on using kinetiq is available in [the wiki](https://github.com/kjkondratuk/kinetiq/wiki)