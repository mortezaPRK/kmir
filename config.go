package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strconv"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kversion"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

var config Config

func initializeConfig() error {
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)
	parser.NamespaceDelimiter = "-"
	topics, err := parser.Parse()
	if err != nil {
		if !flags.WroteHelp(err) {
			return fmt.Errorf("failed to parse flags: %w", err)
		}
	}

	if len(topics) == 0 {
		return fmt.Errorf("no topics specified")
	}

	topicNames := make([]string, 0, len(topics))
	topicOptions := make(map[string]TopicOption, len(topics))
	for _, topic := range topics {
		name, opt, err := toTopic(topic)
		if err != nil {
			return fmt.Errorf("failed to parse topic %q: %w", topic, err)
		}

		topicNames = append(topicNames, name)
		topicOptions[name] = opt
	}

	kVersion := kversion.FromString(opts.KafkaVersion)
	if kVersion == nil {
		return fmt.Errorf("unknown kafka version %q", opts.KafkaVersion)
	}

	sourceOpts, err := toFranzOptions(opts.Source)
	if err != nil {
		return fmt.Errorf("failed to parse source options: %w", err)
	}

	sinkOpts, err := toFranzOptions(opts.Sink)
	if err != nil {
		return fmt.Errorf("failed to parse sink options: %w", err)
	}

	config.Sink = sinkOpts
	config.Source = sourceOpts
	config.ClientID = opts.ClientID
	config.KafkaVersion = kVersion
	config.Topics = topicOptions
	config.TopicNames = topicNames
	config.Timeout = max(opts.Sink.Timeout, opts.Source.Timeout)

	return nil
}

func toFranzOptions(brokerOpts BrokerOptions) ([]kgo.Opt, error) {
	out := make([]kgo.Opt, 0, 4)

	out = append(out, kgo.SeedBrokers(brokerOpts.Brokers...))
	out = append(out, kgo.DialTimeout(brokerOpts.Timeout))

	if brokerOpts.Tls.Enabled {
		tlsConfig := &tls.Config{
			MinVersion:         tls.VersionTLS12,
			Renegotiation:      tls.RenegotiateFreelyAsClient,
			RootCAs:            x509.NewCertPool(),
			InsecureSkipVerify: brokerOpts.Tls.Insecure,
		}

		if brokerOpts.Tls.Cert != "" {
			if ok := tlsConfig.RootCAs.AppendCertsFromPEM([]byte(brokerOpts.Tls.Cert)); !ok {
				return nil, fmt.Errorf("failed to append cert to root CAs")
			}
		}

		out = append(out, kgo.DialTLSConfig(tlsConfig))
	}

	if brokerOpts.Sasl.Enabled {
		if brokerOpts.Sasl.Mechanism == "" || brokerOpts.Sasl.Username == "" || brokerOpts.Sasl.Password == "" {
			return nil, fmt.Errorf("SASL mechanism, username, and password are required")
		}

		var mechanism sasl.Mechanism
		switch brokerOpts.Sasl.Mechanism {
		case "plain":
			mechanism = plain.Auth{
				User: brokerOpts.Sasl.Username,
				Pass: brokerOpts.Sasl.Password,
			}.AsMechanism()
		case "scram-sha-256":
			mechanism = scram.Auth{
				User: brokerOpts.Sasl.Username,
				Pass: brokerOpts.Sasl.Password,
			}.AsSha256Mechanism()
		case "scram-sha-512":
			mechanism = scram.Auth{
				User: brokerOpts.Sasl.Username,
				Pass: brokerOpts.Sasl.Password,
			}.AsSha512Mechanism()
		default:
			return nil, fmt.Errorf("unknown SASL mechanism %q", brokerOpts.Sasl.Mechanism)
		}

		out = append(out, kgo.SASL(mechanism))
	}

	return out, nil
}

func toTopic(value string) (name string, opt TopicOption, err error) {
	parts := strings.Split(value, "@")
	name = parts[0]

	switch len(parts) {
	case 1:
		opt = TopicOption{Offset: -1}
	case 2:
		opt, err = parseTopicOffset(parts[1])
	default:
		err = fmt.Errorf("expected topic@offset or topic@partition:offset,partition:offset... got %q", value)
	}

	return
}

func parseTopicOffset(value string) (TopicOption, error) {
	offsetsOrOffsetPerPartition := strings.Split(value, ",")
	if len(offsetsOrOffsetPerPartition) == 1 {
		offset, err := strconv.ParseInt(offsetsOrOffsetPerPartition[0], 10, 64)
		if err != nil {
			return TopicOption{}, fmt.Errorf("failed to parse offset %q: %w", offsetsOrOffsetPerPartition[0], err)
		}

		return TopicOption{Offset: offset}, nil
	}

	out := TopicOption{
		PerPartitionOffset: map[int32]int64{},
	}
	for _, offsets := range offsetsOrOffsetPerPartition {
		partitionOffset := strings.Split(offsets, ":")
		if len(partitionOffset) != 2 {
			return out, fmt.Errorf("expected partition:offset, got %q", offsets)
		}

		partition, err := strconv.ParseInt(partitionOffset[0], 10, 32)
		if err != nil {
			return out, fmt.Errorf("failed to parse partition %q: %w", partitionOffset[0], err)
		}

		offset, err := strconv.ParseInt(partitionOffset[1], 10, 64)
		if err != nil {
			return out, fmt.Errorf("failed to parse offset %q: %w", partitionOffset[1], err)
		}

		out.PerPartitionOffset[int32(partition)] = offset
	}

	return out, nil
}
