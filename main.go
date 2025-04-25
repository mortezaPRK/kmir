package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/twmb/go-franz/pkg/kadm"
	"github.com/twmb/go-franz/pkg/kgo"
)

func main() {
	if err := initializeConfig(); err != nil {
		slog.Error("Failed to initialize config", slog.Any("error", err))
		return
	}

	var rootCtx = context.Background()

	slog.Info("Creating source Kafka client")
	sourceClient, sourceAdminClient, err := getClients(config.Source)
	if err != nil {
		slog.Error("Failed to create source Kafka client", slog.Any("error", err))
		return
	}
	defer sourceClient.Close()

	slog.Info("Creating sink Kafka client")
	sinkClient, sinkAdminClient, err := getClients(config.Sink)
	if err != nil {
		slog.Error("Failed to create sink Kafka client", slog.Any("error", err))
		return
	}
	defer sinkClient.Close()

	slog.Info("Getting source topics")
	sourceTopics, err := getTopics(rootCtx, sourceAdminClient)
	if err != nil {
		slog.Error("Failed to get source topics", slog.Any("error", err))
		return
	}

	slog.Info("Getted sink topics")
	sinkTopics, err := getTopics(rootCtx, sinkAdminClient)
	if err != nil {
		slog.Error("Failed to get sink topics", slog.Any("error", err))
		return
	}

	slog.Info("Checking source topics")
	if err := checkTopics(sourceTopics); err != nil {
		slog.Error("Failed to check source topics", slog.Any("error", err))
		return
	}

	slog.Info("Deleting existing sink topics")
	if err := deleteExistingTopics(rootCtx, sinkAdminClient, sinkTopics); err != nil {
		slog.Error("Failed to delete existing sink topics", slog.Any("error", err))
		return
	}

	slog.Info("Creating sink topics")
	if err := createTopics(rootCtx, sinkAdminClient, sourceTopics); err != nil {
		slog.Error("Failed to create sink topics", slog.Any("error", err))
		return
	}

	slog.Info("Configuring consumer")
	configureConsumer(sourceClient, sourceTopics)

	slog.Info("Starting mirror")
	for {
		fetches := sourceClient.PollFetches(rootCtx)
		fetches.EachError(func(s string, i int32, err error) {
			slog.LogAttrs(
				rootCtx,
				slog.LevelError,
				"error fetching topic",
				slog.String("topic", s),
				slog.Int("partition", int(i)),
				slog.Any("error", err),
			)
		})

		if fetches.Err() != nil {
			return
		}

		slog.Info("Processing fetches")
		fetches.EachRecord(func(r *kgo.Record) {
			sinkClient.Produce(rootCtx, r, nil)
		})
	}
}

func wait(timeout time.Duration, fn func() bool) error {
	start := time.Now()
	for !fn() {
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for condition")
		}
		time.Sleep(time.Second)
	}
	return nil
}

func getTopics(rootCtx context.Context, client *kadm.Client) (kadm.TopicDetails, error) {
	ctx, cancel := context.WithTimeout(rootCtx, config.Timeout)
	defer cancel()

	sourceTopics, err := client.ListTopics(ctx, config.TopicNames...)
	if err != nil {
		return nil, fmt.Errorf("failed to list source topics: %w", err)
	}

	return sourceTopics, nil
}

func checkTopics(sourceTopics kadm.TopicDetails) error {
	missingTopics := make([]string, 0)

	for topic := range config.Topics {
		if !sourceTopics.Has(topic) {
			missingTopics = append(missingTopics, topic)
		}
	}

	if len(missingTopics) > 0 {
		return fmt.Errorf("source topics missing: %v", missingTopics)
	}
	return nil
}

func deleteExistingTopics(rootCtx context.Context, client *kadm.Client, sinkTopics kadm.TopicDetails) error {
	topicsToDelete := make([]string, 0)

	for topic := range config.Topics {
		if sinkTopics.Has(topic) {
			topicsToDelete = append(topicsToDelete, topic)
		}
	}

	if len(topicsToDelete) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(rootCtx, config.Timeout)
	defer cancel()

	if _, err := client.DeleteTopics(ctx, topicsToDelete...); err != nil {
		return fmt.Errorf("failed to delete topic %v: %w", topicsToDelete, err)
	}

	if err := wait(config.Timeout, func() bool {
		ctx, cancel = context.WithTimeout(rootCtx, config.Timeout)
		defer cancel()

		sinkTopics, err := client.ListTopics(ctx, topicsToDelete...)
		if err != nil {
			slog.Error("Failed to list sink topics to check if they are deleted", slog.Any("error", err))
			return false
		}

		for _, topic := range topicsToDelete {
			if sinkTopics.Has(topic) {
				return false
			}
		}

		return true
	}); err != nil {
		return fmt.Errorf("wait for topics deletion: %w", err)
	}

	return nil
}

func createTopics(rootCtx context.Context, client *kadm.Client, sourceTopics kadm.TopicDetails) error {
	for _, topic := range config.TopicNames {
		if err := createTopic(rootCtx, client, sourceTopics[topic], topic); err != nil {
			return fmt.Errorf("createTopic %q: %w", topic, err)
		}
	}

	if err := wait(config.Timeout, func() bool {
		ctx, cancel := context.WithTimeout(rootCtx, config.Timeout)
		defer cancel()

		sinkTopics, err := client.ListTopics(ctx, config.TopicNames...)
		if err != nil {
			slog.Error("Failed to list sink topics to check if they are created", slog.Any("error", err))
			return false
		}

		for _, topic := range config.TopicNames {
			if !sinkTopics.Has(topic) {
				return false
			}
		}

		return true
	}); err != nil {
		return fmt.Errorf("wait for topics creation: %w", err)
	}

	return nil
}

func createTopic(rootCtx context.Context, client *kadm.Client, dt kadm.TopicDetail, topic string) error {
	ctx, cancel := context.WithTimeout(rootCtx, config.Timeout)
	defer cancel()

	partitions := int32(len(dt.Partitions))

	if _, err := client.CreateTopic(ctx, partitions, -1, nil, topic); err != nil {
		return fmt.Errorf("failed to create topic %q: %w", topic, err)
	}
	return nil
}

func configureConsumer(client *kgo.Client, sourceTopics kadm.TopicDetails) {
	partitions := map[string]map[int32]kgo.Offset{}
	for topic, dt := range sourceTopics {
		cfg := config.Topics[topic]
		offsetCfg := map[int32]kgo.Offset{}

		for _, partition := range dt.Partitions {
			if offset, ok := cfg.OffsetOf(partition.Partition); ok {
				offsetCfg[partition.Partition] = kgo.NewOffset().At(offset)
			}
		}

		partitions[topic] = offsetCfg
	}

	client.AddConsumePartitions(partitions)
}

func getClients(opts []kgo.Opt) (*kgo.Client, *kadm.Client, error) {
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create source admin client: %w", err)
	}

	return client, kadm.NewClient(client), nil
}
