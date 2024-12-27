package rank

import (
	"context"
)

type IntegralRankIFace interface {
	// IntegralRankMonthList 积分排行榜月
	IntegralRankMonthList(ctx context.Context, req *IntegralRankListRequest) (*IntegralRankListResponse, error)

	// ListUserInfoByIds 查询用户信息，带缓存
	ListUserInfoByIds(ctx context.Context, userID []uint64) (map[uint64]*UserInfo, error)

	// IntegralChangeHandle 事件更新缓存 （更新每月实时排行榜）
	IntegralChangeHandle(ctx context.Context, opt *IntegralChange) error

	// Flush 将数据 flush into db
	Flush(ctx context.Context, yearMonth string, force bool, listId uint64) error

	FlushAll(ctx context.Context, yearMonth string, force bool) error
}
