# Kafka Mirror CLI

This application mirrors Kafka topics from a cluster to your local machine as part of your development environment. 

**Note:** This app is only intended for development purposes and should not be used for any kind of serious work.

## Installation

You can download the CLI from the GitHub release page or Docker Hub (mortezaprk/kmir).

## Usage

```sh
Usage:
  kmir [OPTIONS] TOPICS...

Application Options:
      --client-id              Client ID [$CLIENT_ID]
      --kafka-version          Kafka version [$KAFKA_VERSION]

Source:
      --source-brokers         Comma-separated list of Kafka brokers [$SOURCE_BROKERS]
      --source-timeout         Timeout for Kafka (default: 10s) [$SOURCE_TIMEOUT]

    TLS:
        --source-tls-enabled     Enable TLS [$SOURCE_TLS_ENABLED]
        --source-tls-cert        TLS certificate [$SOURCE_TLS_CERT]
        --source-tls-insecure    Skip TLS verification [$SOURCE_TLS_INSECURE]

    SASL:
        --source-sasl-enabled    Enable SASL [$SOURCE_SASL_ENABLED]
        --source-sasl-username   SASL username [$SOURCE_SASL_USERNAME]
        --source-sasl-password   SASL password [$SOURCE_SASL_PASSWORD]
        --source-sasl-mechanism  SASL mechanism [$SOURCE_SASL_MECHANISM]

Sink:
      --sink-brokers           Comma-separated list of Kafka brokers [$SINK_BROKERS]
      --sink-timeout           Timeout for Kafka (default: 10s) [$SINK_TIMEOUT]

    TLS:
        --sink-tls-enabled       Enable TLS [$SINK_TLS_ENABLED]
        --sink-tls-cert          TLS certificate [$SINK_TLS_CERT]
        --sink-tls-insecure      Skip TLS verification [$SINK_TLS_INSECURE]

    SASL:
        --sink-sasl-enabled      Enable SASL [$SINK_SASL_ENABLED]
        --sink-sasl-username     SASL username [$SINK_SASL_USERNAME]
        --sink-sasl-password     SASL password [$SINK_SASL_PASSWORD]
        --sink-sasl-mechanism    SASL mechanism [$SINK_SASL_MECHANISM]

Help Options:
  -h, --help                   Show this help message
```

### Topics

The positional arguments, which specify the topic information, can be in any of the following forms:
- `topic_name`: Mirrors the topic from the end.
- `topic_name@offset`: Mirrors the topic and all partitions starting from the given offset.
- `topic_name@partition:offset,partition:offset`: Only the mentioned partitions are mirrored from the given offset.

> Since this app uses `go-franz` internally, two offset values have specific meanings:
> - `-1`: From the end.
> - `-2` or lower: From the start.

### Example

```sh
kmir --source-brokers=localhost:9092 --sink-brokers=localhost:9093 --client-id=my-client --kafka-version=2.7.0 topic1 topic2@-1 topic3@0:100,1:200
```
