package rank_kit

import (
	"context"
)

type RankOrder uint32

type RankWriter interface {
	// StoreRankItem 单个存储排行成员项
	StoreRankItem(ctx context.Context, rankListID RankListID, item *RankZItem, scope string) error
	// BulkStoreRankItem 批存储排行成员项
	BulkStoreRankItem(ctx context.Context, rankListID RankListID, items []*RankZItem, scope string) (int64, error)
	// RemRankByItemId 移除排行成员
	RemRankByItemId(ctx context.Context, rankListID RankListID, id RankItemID, scope string) error
	// RemRankByItemIds 批量溢出排行榜成员
	RemRankByItemIds(ctx context.Context, rankListID RankListID, ids []string) error
}

type RankReader interface {
	// GetRankItemById 获取排行成员项
	GetRankItemById(ctx context.Context, rankListID RankListID, id RankItemID, scope string) (*RankZItem, error)
	// GetRankByItemId 获取排行成员项标识的排名
	GetRankByItemId(ctx context.Context, rankListID RankListID, id RankItemID, order RankOrder) (int64, error)
	// GetScoreByItemId 获取成员的分值
	GetScoreByItemId(ctx context.Context, rankListID RankListID, id RankItemID) (int64, error)
	// RankList 查询排行榜
	RankList(ctx context.Context, rankListID RankListID, order RankOrder, offset, limit int64, scope string) ([]*RankZItem, error)
	// GetRankItemCount 获取排行榜成员项总数量
	GetRankItemCount(ctx context.Context, rankListID RankListID) (int64, error)
	// GetScoreWinRate 获取某个分值的击败排名比例
	GetScoreWinRate(ctx context.Context, rankListID RankListID, score int64, order RankOrder) (float64, error)
}

type ScoreRegister interface {
	Register(Score RankScoreIFace) error
}

type RankStorageIFace interface {
	RankWriter
	RankReader
	ScoreRegister
}
