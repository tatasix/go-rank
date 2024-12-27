package rank_kit

import (
	"context"
)

type RankRebuildIFace interface {
	Rebuild(ctx context.Context, RankListID RankListID) (int, error)
	RebuildAll(ctx context.Context) (int, error)
}
