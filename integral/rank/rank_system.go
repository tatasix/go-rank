package rank

import (
	"context"
	"github.com/gomodule/redigo/redis"
	"time"

	"demo/lock"
	"demo/rank/rank_kit"
)

var _ rank_kit.RankSystemIFace = (*RankSystem)(nil)

type RankSystem struct {
	rank_kit.RankSourceIFace
	rank_kit.RankStorageIFace
	rank_kit.RankRebuildIFace
	rank_kit.RankScoreIFace
}

func NewRankSystem() (rank_kit.RankSystemIFace, error) {
	ranksource := &RankSource{}
	// 注入依赖，应该用IOC解决
	rankScore := NewZSetRankScore(DescTimeOrder)
	redisPool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp",
				"redis host",
				redis.DialPassword("redis password"),
			)
		},
	}

	storage := NewRedisStorage(redisPool, rankScore)
	locker := NewLock()
	rankBuilder, err := NewRankRebuild(ranksource, storage, 1, 4000, locker)
	if err != nil {
		return nil, err
	}
	// storage.SetScope(scope)
	return &RankSystem{
		RankSourceIFace:  ranksource,
		RankStorageIFace: storage,
		RankRebuildIFace: rankBuilder,
		RankScoreIFace:   rankScore,
	}, nil
}

type RankLock struct {
	Locker lock.RedisLockIFace
}

func NewLock() *RankLock {
	backoff := lock.NewExponentialBackoff(
		time.Duration(20)*time.Millisecond,
		time.Duration(1000)*time.Millisecond,
	)
	redisPool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp",
				"redis host",
				redis.DialPassword("redis password"),
			)
		},
	}

	redisLock := lock.NewRedisLock(redisPool, lock.WithBackoff(backoff))
	return &RankLock{
		Locker: redisLock,
	}
}

func (l *RankLock) Lock(ctx context.Context, key string) (lock.LockInstanceIFace, error) {
	var rebuildLockInstance lock.LockInstanceIFace = lock.NewLockInstance(l.Locker)
	if err := rebuildLockInstance.MustSet(ctx, key); err != nil {
		if err == lock.ErrLockFail {
			return nil, ErrRankListRebuilding
		}
		return nil, err
	}
	return rebuildLockInstance, nil
}
