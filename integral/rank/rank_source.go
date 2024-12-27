package rank

import (
	"context"
	"demo/rank/rank_kit"
	"strconv"
)

type RankSource struct {
}

func NewRankSource() *RankSource {
	return &RankSource{}
}

// RankSourceItem 获取某个排行榜单项的数据
func (f *RankSource) RankSourceItem(ctx context.Context, rankListId rank_kit.RankListID, rankItemid rank_kit.RankItemID) (*rank_kit.RankZItem, error) {
	panic("not implemented")

}

// RankSourceRankList 获取排行榜列表
func (f *RankSource) RankSourceRankList(ctx context.Context, offset int64, limit int64) ([]rank_kit.RankListID, error) {

	//从db获取数据
	//repo.RankSourceList(ctx, uint64(listId), startTime, endTime, offset, limit, 1)

	return nil, nil
}

// RankSourceList 获取某个排行榜真实数据源
func (f *RankSource) RankSourceList(ctx context.Context, rankListId rank_kit.RankListID, offset int64, limit int64) ([]*rank_kit.RankZItem, error) {

	//从db获取数据
	//repo.RankSourceList(ctx, uint64(listId), startTime, endTime, offset, limit, 1)

	return nil, nil
}

func getUserId(itemID rank_kit.RankItemID) (uint64, error) {
	userID, err := strconv.Atoi(string(itemID))
	if err != nil {
		return 0, err
	}
	return uint64(userID), nil
}
