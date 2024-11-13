package main

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

var ctx = context.Background()

func main() {
	// Parse flags
	cfg, err := parseFlags()
	if err != nil {
		panic(fmt.Errorf("failed to parse flags: %w", err))
	}

	// Create source and sink clients
	sourceClient, err := kgo.NewClient(cfg.Source...)
	if err != nil {
		panic(fmt.Errorf("failed to create source admin client: %w", err))
	}
	defer sourceClient.Close()

	sinkClient, err := kgo.NewClient(cfg.Sink...)
	if err != nil {
		panic(fmt.Errorf("failed to create sink admin client: %w", err))
	}
	defer sinkClient.Close()

	// Create admin clients
	sourceAdminClient := kadm.NewClient(sourceClient)
	sinkAdminClient := kadm.NewClient(sinkClient)

	// List topics in source and sink
	topicNames := make([]string, 0, len(cfg.Topics))
	for topic := range cfg.Topics {
		topicNames = append(topicNames, topic)
	}

	sourceTopics, err := sourceAdminClient.ListTopics(ctx, topicNames...)
	if err != nil {
		panic(fmt.Errorf("failed to list source topics: %w", err))
	}

	sinkTopics, err := sinkAdminClient.ListTopics(ctx, topicNames...)
	if err != nil {
		panic(fmt.Errorf("failed to list sink topics: %w", err))
	}

	// Determine topics to delete and check for missing topics
	missingTopics := make([]string, 0)
	topicsToDelete := make([]string, 0)

	for topic := range cfg.Topics {
		if !sourceTopics.Has(topic) {
			missingTopics = append(missingTopics, topic)
		}
		if sinkTopics.Has(topic) {
			topicsToDelete = append(topicsToDelete, topic)
		}
	}

	// Ensure all source topics exist
	if len(missingTopics) > 0 {
		panic(fmt.Errorf("source topics missing: %v", missingTopics))
	}

	// delete topics from sink
	if len(topicsToDelete) > 0 {
		if _, err := sinkAdminClient.DeleteTopics(ctx, topicsToDelete...); err != nil {
			panic(fmt.Errorf("failed to delete topic %v: %w", topicsToDelete, err))
		}
		fmt.Println("Deleted topics:", topicsToDelete)
	}

	// create topics in sink
	for _, topic := range topicNames {
		dt := sourceTopics[topic]
		if _, err := sinkAdminClient.CreateTopic(
			ctx,
			int32(len(dt.Partitions)),
			1,
			nil,
			topic,
		); err != nil {
			panic(fmt.Errorf("failed to create topic %q: %w", topic, err))
		}

		fmt.Println("Created topic:", topic)
	}

	// Start consuming and producing to same partition topic
	partitions := map[string]map[int32]kgo.Offset{}
	for topic, dt := range sourceTopics {
		if _, exists := partitions[topic]; !exists {
			partitions[topic] = map[int32]kgo.Offset{}
		}

		offset := kgo.NewOffset().AtEnd()
		if cfg.Topics[topic].FromStart {
			offset = kgo.NewOffset().AtStart()
		}

		for _, partition := range dt.Partitions {
			partitions[topic][partition.Partition] = offset
		}
	}

	sourceClient.AddConsumePartitions(partitions)

	// Fetch time :)
	for {
		fetches := sourceClient.PollFetches(ctx)
		fetches.EachError(func(s string, i int32, err error) {
			fmt.Printf("error fetching %s:%d: %v\n", s, i, err)
		})
		fetches.EachRecord(func(r *kgo.Record) {
			sinkClient.Produce(ctx, r, nil)
			fmt.Printf("produced record to %s:%d@%d\n", r.Topic, r.Partition, r.Offset)
		})
	}
}
