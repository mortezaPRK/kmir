package main

import (
	"testing"
)

func TestTopicOption_OffsetOf(t *testing.T) {
	tests := []struct {
		name      string
		opt       TopicOption
		partition int32
		want      int64
		wantOk    bool
	}{
		{
			name:      "returns offset when PerPartitionOffset is nil",
			opt:       TopicOption{Offset: 100},
			partition: 0,
			want:      100,
			wantOk:    true,
		},
		{
			name: "returns offset when partition exists in PerPartitionOffset",
			opt: TopicOption{
				Offset:             0,
				PerPartitionOffset: map[int32]int64{0: 50, 1: 100},
			},
			partition: 1,
			want:      100,
			wantOk:    true,
		},
		{
			name: "returns not found when partition doesn't exist in PerPartitionOffset",
			opt: TopicOption{
				Offset:             0,
				PerPartitionOffset: map[int32]int64{0: 50},
			},
			partition: 1,
			want:      0,
			wantOk:    false,
		},
		{
			name:      "returns zero offset with empty PerPartitionOffset",
			opt:       TopicOption{PerPartitionOffset: map[int32]int64{}},
			partition: 0,
			want:      0,
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := tt.opt.OffsetOf(tt.partition)
			if got != tt.want {
				t.Errorf("OffsetOf() got = %v, want %v", got, tt.want)
			}
			if gotOk != tt.wantOk {
				t.Errorf("OffsetOf() gotOk = %v, wantOk %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestTopicOption_OffsetOf_MultiplePartitions(t *testing.T) {
	opt := TopicOption{
		PerPartitionOffset: map[int32]int64{
			0: 100,
			1: 200,
			2: 300,
		},
	}

	tests := []struct {
		partition int32
		want      int64
		wantOk    bool
	}{
		{0, 100, true},
		{1, 200, true},
		{2, 300, true},
		{3, 0, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, gotOk := opt.OffsetOf(tt.partition)
			if got != tt.want {
				t.Errorf("OffsetOf() partition %d got = %v, want %v", tt.partition, got, tt.want)
			}
			if gotOk != tt.wantOk {
				t.Errorf("OffsetOf() partition %d gotOk = %v, wantOk %v", tt.partition, gotOk, tt.wantOk)
			}
		})
	}
}

func TestTopicOption_OffsetOf_NegativeOffset(t *testing.T) {
	opt := TopicOption{
		Offset: -1, // Represents "latest" or "end"
	}

	got, gotOk := opt.OffsetOf(0)
	if got != -1 {
		t.Errorf("OffsetOf() got = %v, want -1", got)
	}
	if !gotOk {
		t.Errorf("OffsetOf() gotOk = false, want true")
	}
}
