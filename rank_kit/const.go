package rank_kit

import "errors"

// ZSetKey 集合key
type ZSetKey string

type RankListID string

// RankItemID 集合内部Item标识
type RankItemID string

type RankZItem struct {
	ItemContext
	Score int64 // 权重
	Time  int64 // 时间
}

func NewRankItem(rankItemId RankItemID, rankScore int64, createTime int64, context map[string]string) *RankZItem {
	return &RankZItem{
		ItemContext: ItemContext{
			RankItemID: rankItemId,
			Context:    context,
		},
		Score: rankScore,
		Time:  createTime,
	}
}

// ItemContext 排行上下文
type ItemContext struct {
	RankItemID RankItemID        // 标识 		-- 比如用户ID (唯一表示)
	Context    map[string]string // 上下文信息 -- 比如用户头像，姓名
}

func (i *RankZItem) Get(key string) interface{} {
	if i == nil {
		return nil
	}
	if v, ok := i.Context[key]; ok {
		return v
	}
	return nil
}

var (
	ErrRankStorageNotFoundItem  = errors.New("未查到排行项")
	ErrRankStorageFindRankItem  = errors.New("获取排行项排名发生错误")
	ErrRankStorageStoreRankItem = errors.New("保存排行项发生错误")
)

var (
	AscRankOrder  RankOrder = 1 // 正序排行
	DescRankOrder RankOrder = 2 // 倒序排行
)
