package rank

import (
	"context"
	rank_kit2 "demo/rank/rank_kit"
)

const (
	max_time = 2524579200 // 2050/1/1 0:0:0
)

var (
	pos1Size       = 31
	pos2Size       = 32
	pos1Max  int64 = 1<<uint(pos1Size) - 1 // later go 1.13 可以不用uint
	pos2Max  int64 = 1<<uint(pos2Size) - 1
)

const (
	DescTimeOrder    = 1
	AscTimeOrder     = 2
	DefaultTimeOrder = DescTimeOrder
)

var _ rank_kit2.RankScoreIFace = (*ZSetRankScore)(nil)

// ZSetRankScore 设计思路
// 1. 首位标志位不用 高31位存储分数 低32位存储时间
// 2. 如果时间倒序 则直接存储时间
// 3. 如果时间正序 则直接MAX_TIME-时间
type ZSetRankScore struct {
	TimeOrder uint32
}

func NewZSetRankScore(timeOrder uint32) *ZSetRankScore {

	zs := &ZSetRankScore{
		TimeOrder: DefaultTimeOrder,
	}

	if timeOrder != zs.TimeOrder {
		switch timeOrder {
		case DescTimeOrder, AscTimeOrder:
			zs.TimeOrder = timeOrder
		}
	}

	return zs
}

func (z *ZSetRankScore) RankItemReScore(ctx context.Context, item *rank_kit2.RankZItem) (*rank_kit2.RankZItem, error) {

	tmp := *(item)

	tmp.Score = genScore(item.Score, genTimeScore(z.TimeOrder, int64(item.Time)))

	return &tmp, nil
}

func (z *ZSetRankScore) RankItemDeScore(ctx context.Context, item *rank_kit2.RankZItem) (*rank_kit2.RankZItem, error) {

	tmp := *(item)
	score := tmp.Score
	tmp.Score = deScore(score)
	tmp.Time = deCreateTime(score, z.TimeOrder)

	return &tmp, nil
}

func (z *ZSetRankScore) ReScore(ctx context.Context, score int64, createTime int64) int64 {
	return genScore(score, createTime)
}

func (z *ZSetRankScore) DeScore(ctx context.Context, score int64) (int64, int64) {
	return deScore(score), deCreateTime(score, z.TimeOrder)
}

func deScore(encScore int64) int64 {
	return encScore >> uint(pos2Size)
}

func deCreateTime(encScore int64, timeOrder uint32) int64 {

	wTime := encScore & pos2Max

	switch timeOrder {
	case DescTimeOrder:
		return wTime
	case AscTimeOrder:
		return max_time - wTime
	}

	return 0
}

func genScore(gameScore int64, timeScore int64) int64 {
	return ((gameScore & pos1Max) << uint(pos2Size)) | (timeScore & pos2Max)
}

func genTimeScore(timeOrder uint32, createTime int64) int64 {

	switch timeOrder {
	case DescTimeOrder:
		return createTime
	case AscTimeOrder:
		return max_time - createTime
	}

	return 0
}
