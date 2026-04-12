package main

import (
	"strings"
	"testing"
)

func TestParseTopicOffset(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    TopicOption
		wantErr bool
	}{
		{
			name:    "single offset - positive",
			value:   "100",
			want:    TopicOption{Offset: 100},
			wantErr: false,
		},
		{
			name:    "single offset - zero",
			value:   "0",
			want:    TopicOption{Offset: 0},
			wantErr: false,
		},
		{
			name:    "single offset - negative",
			value:   "-1",
			want:    TopicOption{Offset: -1},
			wantErr: false,
		},
		{
			name:    "single partition offset (comma required for multiple)",
			value:   "0:100",
			want:    TopicOption{},
			wantErr: true,
		},
		{
			name:  "multiple partition offsets",
			value: "0:100,1:200,2:300",
			want: TopicOption{
				PerPartitionOffset: map[int32]int64{0: 100, 1: 200, 2: 300},
			},
			wantErr: false,
		},
		{
			name:    "invalid offset - non-numeric",
			value:   "abc",
			want:    TopicOption{},
			wantErr: true,
		},
		{
			name:  "invalid partition offset - missing colon",
			value: "0-100",
			want: TopicOption{
				PerPartitionOffset: map[int32]int64{},
			},
			wantErr: true,
		},
		{
			name:  "invalid partition offset - invalid partition",
			value: "abc:100",
			want: TopicOption{
				PerPartitionOffset: map[int32]int64{},
			},
			wantErr: true,
		},
		{
			name:  "invalid partition offset - invalid offset",
			value: "0:abc",
			want: TopicOption{
				PerPartitionOffset: map[int32]int64{},
			},
			wantErr: true,
		},
		{
			name:  "invalid partition offset - three parts",
			value: "0:100:200",
			want: TopicOption{
				PerPartitionOffset: map[int32]int64{},
			},
			wantErr: true,
		},
		{
			name:  "partition offsets with negative values",
			value: "0:-1,1:0",
			want: TopicOption{
				PerPartitionOffset: map[int32]int64{0: -1, 1: 0},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTopicOffset(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTopicOffset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Offset != tt.want.Offset {
					t.Errorf("parseTopicOffset() Offset = %v, want %v", got.Offset, tt.want.Offset)
				}
				if len(got.PerPartitionOffset) != len(tt.want.PerPartitionOffset) {
					t.Errorf("parseTopicOffset() PerPartitionOffset length = %v, want %v", len(got.PerPartitionOffset), len(tt.want.PerPartitionOffset))
				}
				for k, v := range tt.want.PerPartitionOffset {
					if got.PerPartitionOffset[k] != v {
						t.Errorf("parseTopicOffset() PerPartitionOffset[%d] = %v, want %v", k, got.PerPartitionOffset[k], v)
					}
				}
			}
		})
	}
}

func TestToTopic(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantOpt TopicOption
		wantErr bool
	}{
		{
			name:    "topic only",
			value:   "my-topic",
			want:    "my-topic",
			wantOpt: TopicOption{Offset: -1},
			wantErr: false,
		},
		{
			name:    "topic with offset",
			value:   "my-topic@100",
			want:    "my-topic",
			wantOpt: TopicOption{Offset: 100},
			wantErr: false,
		},
		{
			name:    "topic with partition offsets",
			value:   "my-topic@0:100,1:200",
			want:    "my-topic",
			wantOpt: TopicOption{PerPartitionOffset: map[int32]int64{0: 100, 1: 200}},
			wantErr: false,
		},
		{
			name:    "topic with dash",
			value:   "my-topic-123",
			want:    "my-topic-123",
			wantOpt: TopicOption{Offset: -1},
			wantErr: false,
		},
		{
			name:    "topic with offset zero",
			value:   "my-topic@0",
			want:    "my-topic",
			wantOpt: TopicOption{Offset: 0},
			wantErr: false,
		},
		{
			name:    "invalid - multiple at signs",
			value:   "my-topic@100@200",
			want:    "my-topic",
			wantOpt: TopicOption{},
			wantErr: true,
		},
		{
			name:    "empty topic name with offset (valid by parsing logic)",
			value:   "@100",
			want:    "",
			wantOpt: TopicOption{Offset: 100},
			wantErr: false,
		},
		{
			name:    "invalid - bad offset format",
			value:   "my-topic@abc",
			want:    "my-topic",
			wantOpt: TopicOption{},
			wantErr: true,
		},
		{
			name:    "topic with underscores",
			value:   "my_topic_123",
			want:    "my_topic_123",
			wantOpt: TopicOption{Offset: -1},
			wantErr: false,
		},
		{
			name:    "topic with dots",
			value:   "my.topic.name",
			want:    "my.topic.name",
			wantOpt: TopicOption{Offset: -1},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOpt, err := toTopic(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("toTopic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("toTopic() got = %v, want %v", got, tt.want)
			}
			if !tt.wantErr {
				if gotOpt.Offset != tt.wantOpt.Offset {
					t.Errorf("toTopic() Offset = %v, want %v", gotOpt.Offset, tt.wantOpt.Offset)
				}
				if len(gotOpt.PerPartitionOffset) != len(tt.wantOpt.PerPartitionOffset) {
					t.Errorf("toTopic() PerPartitionOffset length = %v, want %v", len(gotOpt.PerPartitionOffset), len(tt.wantOpt.PerPartitionOffset))
				}
				for k, v := range tt.wantOpt.PerPartitionOffset {
					if gotOpt.PerPartitionOffset[k] != v {
						t.Errorf("toTopic() PerPartitionOffset[%d] = %v, want %v", k, gotOpt.PerPartitionOffset[k], v)
					}
				}
			}
		})
	}
}

func TestToTopic_LongTopicNames(t *testing.T) {
	longTopic := strings.Repeat("a", 249) + "@100"
	got, gotOpt, err := toTopic(longTopic)

	if err != nil {
		t.Errorf("toTopic() unexpected error: %v", err)
	}
	if len(got) != 249 {
		t.Errorf("toTopic() got topic length = %v, want 249", len(got))
	}
	if gotOpt.Offset != 100 {
		t.Errorf("toTopic() Offset = %v, want 100", gotOpt.Offset)
	}
}

func BenchmarkParseTopicOffset(b *testing.B) {
	tests := []string{
		"100",
		"0:100,1:200,2:300",
		"0:100,1:200,2:300,3:400,4:500,5:600,6:700,7:800",
	}

	for _, tt := range tests {
		b.Run(tt, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				parseTopicOffset(tt)
			}
		})
	}
}

func BenchmarkToTopic(b *testing.B) {
	tests := []string{
		"my-topic",
		"my-topic@100",
		"my-topic@0:100,1:200,2:300",
	}

	for _, tt := range tests {
		b.Run(tt, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				toTopic(tt)
			}
		})
	}
}
