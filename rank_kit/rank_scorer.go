package rank_kit

import (
	"context"
)

type RankScoreIFace interface {
	// RankItemReScore 排行项重算分数
	RankItemReScore(ctx context.Context, item *RankZItem) (*RankZItem, error)
	// RankItemDeScore 排行项反算分数
	RankItemDeScore(ctx context.Context, item *RankZItem) (*RankZItem, error)
	// ReScore 重算分数
	ReScore(ctx context.Context, score int64, createTime int64) int64
	// DeScore 反算分数
	DeScore(ctx context.Context, encScore int64) (int64, int64)
}
