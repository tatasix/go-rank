package rank_kit

import (
	"context"
)

type RankSourceIFace interface {
	// RankSourceItem 获取某个排行榜单项的数据
	RankSourceItem(ctx context.Context, rankListId RankListID, rankItemid RankItemID) (*RankZItem, error)
	// RankSourceRankList 获取排行榜列表
	RankSourceRankList(ctx context.Context, offset, limit int64) ([]RankListID, error)
	// RankSourceList 获取某个排行榜真实数据源
	RankSourceList(ctx context.Context, rankListId RankListID, offset, limit int64) ([]*RankZItem, error)
}
