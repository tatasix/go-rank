package rank

import (
	"demo/rank/rank_kit"
	"encoding/json"
)

func IntegralRankFromZItem(yearMonth string, listId uint64, zItem *rank_kit.RankZItem) (*IntegralRank, error) {
	userID, err := getUserId(zItem.RankItemID)
	if err != nil {
		return nil, err
	}

	context, err := json.Marshal(zItem.Context)
	if err != nil {
		return nil, err
	}

	return &IntegralRank{
		UserID:      userID,
		ListId:      listId,
		YearMonth:   yearMonth,
		AddIntegral: uint64(zItem.Score),
		Context:     string(context),
		RecordTime:  uint32(zItem.Time),
	}, nil
}
