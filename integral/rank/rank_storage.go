package rank

import (
	"context"
	"demo/rank/rank_kit"
	"fmt"
	"strconv"

	"encoding/json"
	"github.com/gomodule/redigo/redis"
)

var _ rank_kit.RankStorageIFace = (*RedisStorage)(nil)
var _ RedisStorageIFace = (*RedisStorage)(nil)

const DEFAULT_PREFIX = "tata:rank"
const EXPIRED_DAY = 0

type RedisStorageIFace interface {
	SetExpire(expireDay int64)
	SetPrefix(prefix string)
}

type RedisStorage struct {
	RankScore  rank_kit.RankScoreIFace
	redisPool  *redis.Pool
	Prefix     string
	ExpiredDay int64  // 存储有效期
	Scope      string // not safe
}

func NewRedisStorage(redisConn *redis.Pool, Score rank_kit.RankScoreIFace) *RedisStorage {
	storage := &RedisStorage{
		redisPool:  redisConn,
		RankScore:  Score,
		Prefix:     DEFAULT_PREFIX,
		ExpiredDay: EXPIRED_DAY,
	}
	storage.Register(Score)
	storage.Scope = "all"
	return storage
}

// StoreRankItem 单个存储排行成员项
func (r *RedisStorage) StoreRankItem(ctx context.Context, rankListID rank_kit.RankListID, item *rank_kit.RankZItem, scope string) error {
	if item == nil {
		return nil
	}
	// re score
	newItem, err := r.RankScore.RankItemReScore(ctx, item)
	if err != nil {
		return err
	}
	item = newItem

	// start store
	conn, _ := r.redisPool.GetContext(ctx)
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Println(fmt.Sprintf("redis conn close err:%s", err.Error()))
		}
	}()

	zk := r.zsetPrefix(rankListID, scope)
	zm := r.itemKey(string(item.RankItemID))

	ctxKey := r.hashPrefix(rankListID, scope)
	ctxField := r.itemKey(string(item.RankItemID))

	_, err = conn.Do("ZADD", zk, item.Score, zm)
	if err != nil {
		fmt.Println(fmt.Sprintf("StoreRankZItem redis ZADD err:%s", err.Error()))
		return rank_kit.ErrRankStorageStoreRankItem
	}
	context, err := json.Marshal(item.Context)
	if err != nil {
		return err
	}
	_, err = conn.Do("HSET", ctxKey, ctxField, context)
	if err != nil {
		fmt.Println(fmt.Sprintf("StoreRankZItem redis HSET err:%s", err.Error()))
		return rank_kit.ErrRankStorageStoreRankItem
	}

	return nil
}

// BulkStoreRankItem 批存储排行成员项
func (r *RedisStorage) BulkStoreRankItem(ctx context.Context, rankListID rank_kit.RankListID, inputItems []*rank_kit.RankZItem, scope string) (int64, error) {

	if len(inputItems) == 0 {
		return 0, nil
	}

	// re score
	items := make([]*rank_kit.RankZItem, 0, len(inputItems))
	for _, item := range inputItems {
		newItem, err := r.RankScore.RankItemReScore(ctx, item)
		if err != nil {
			return 0, err
		}
		items = append(items, newItem)
	}

	// start store
	conn, _ := r.redisPool.GetContext(ctx)
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Println(fmt.Sprintf("redis conn close err:%s", err.Error()))
		}
	}()

	// rank list zset key redis key
	zk := r.zsetPrefix(rankListID, scope)
	// redis key
	ctxKey := r.hashPrefix(rankListID, scope)
	hashParams := make([]interface{}, 0, len(items)*2+1)
	hashParams = append(hashParams, ctxKey)
	zAddParams := make([]interface{}, 0, len(items)*2+1)
	zAddParams = append(zAddParams, zk)
	for _, item := range items {
		// zset member key
		zmk := r.itemKey(string(item.RankItemID))
		zAddParams = append(zAddParams, strconv.Itoa(int(item.Score)), zmk)
		// hash field
		field := r.itemKey(string(item.RankItemID))

		hashParams = append(hashParams, field, JsonMarshal(item.Context))
	}

	_, err := conn.Do("ZADD", zAddParams...)
	if err != nil {
		fmt.Println(fmt.Sprintf("StoreRankZItem redis ZADD err:%s", err.Error()))
		return 0, rank_kit.ErrRankStorageStoreRankItem
	}

	_, err = conn.Do("HMSET", hashParams...)
	if err != nil {
		fmt.Println(fmt.Sprintf("StoreRankZItem redis HSET err:%s", err.Error()))
		return 0, rank_kit.ErrRankStorageStoreRankItem
	}

	return 0, nil
}

// RemRankByItemId 移除排行成员
func (r *RedisStorage) RemRankByItemId(ctx context.Context, rankListID rank_kit.RankListID, id rank_kit.RankItemID, scope string) error {
	conn := r.redisPool.Get()
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Println(fmt.Sprintf("redis conn close err:%s", err.Error()))
		}
	}()

	zk := r.zsetPrefix(rankListID, scope)
	zm := r.itemKey(string(id))

	n, err := redis.Int(conn.Do("ZREM", zk, zm))
	if err != nil {
		return err
	}

	if n == 0 {
		return rank_kit.ErrRankStorageNotFoundItem
	}

	return nil
}

// RemRankByItemIds 批量移除排行榜成员
func (r *RedisStorage) RemRankByItemIds(ctx context.Context, rankListID rank_kit.RankListID, ids []string) error {
	panic("not implemented") // TODO: Implement
}

// GetRankItemById 获取排行成员项
func (r *RedisStorage) GetRankItemById(ctx context.Context, rankListID rank_kit.RankListID, id rank_kit.RankItemID, scope string) (*rank_kit.RankZItem, error) {
	conn, _ := r.redisPool.GetContext(ctx)
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Println(fmt.Sprintf("redis conn close err:%s", err.Error()))
		}
	}()

	encScore, err := r.getRawScore(ctx, rankListID, id, scope)
	if err != nil {
		return nil, err
	}

	score, createTime := r.RankScore.DeScore(ctx, encScore)

	ctxStr, err := r.getRankItemCtx(ctx, conn, rankListID, id, scope)
	if err != nil {
		return nil, err
	}

	ri := rank_kit.NewRankItem(id, score, createTime, ctxStr)

	return ri, nil
}

// GetRankByItemId 获取排行成员项标识的排名
func (r *RedisStorage) GetRankByItemId(ctx context.Context, rankListID rank_kit.RankListID, id rank_kit.RankItemID, order rank_kit.RankOrder) (int64, error) {
	panic("not implemented") // TODO: Implement
}

// GetScoreByItemId 获取成员的分值
func (r *RedisStorage) GetScoreByItemId(ctx context.Context, rankListID rank_kit.RankListID, id rank_kit.RankItemID) (int64, error) {
	panic("not implemented") // TODO: Implement
}

// RankList 查询排行榜
func (r *RedisStorage) RankList(ctx context.Context, rankListID rank_kit.RankListID, order rank_kit.RankOrder, offset int64, limit int64, scope string) ([]*rank_kit.RankZItem, error) {

	conn, _ := r.redisPool.GetContext(ctx)
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Println(fmt.Sprintf("redis conn close err:%s", err.Error()))
		}
	}()

	zk := r.zsetPrefix(rankListID, scope)

	var reply []string
	var err error
	switch order {
	case rank_kit.AscRankOrder:
		reply, err = redis.Strings(conn.Do("ZRANGE", zk, offset, limitToStop(offset, limit), "WITHSCORES"))
	case rank_kit.DescRankOrder:
		reply, err = redis.Strings(conn.Do("ZREVRANGE", zk, offset, limitToStop(offset, limit), "WITHSCORES"))
	}

	if err != nil {
		return nil, rank_kit.ErrRankStorageFindRankItem
	}

	res, err := formatRankItemFromReplyStrings(reply)
	if err != nil {
		return nil, err
	}

	// de score and get ctx
	rankItems := make([]rank_kit.RankItemID, 0, len(res))
	for i, item := range res {

		newItem, err := r.RankScore.RankItemDeScore(ctx, item)
		if err != nil {
			return nil, err
		}

		res[i] = newItem

		rankItems = append(rankItems, item.RankItemID)
	}

	ctxArr, err := r.multiGetRankItemCtx(ctx, conn, rankListID, rankItems, scope)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(res); i++ {
		res[i].Context = ctxArr[i]
	}

	return res, nil
}

// GetRankItemCount 获取排行榜成员项总数量
func (r *RedisStorage) GetRankItemCount(ctx context.Context, rankListID rank_kit.RankListID) (int64, error) {
	panic("not implemented") // TODO: Implement
}

// GetScoreWinRate 获取某个分值的击败排名比例
func (r *RedisStorage) GetScoreWinRate(ctx context.Context, rankListID rank_kit.RankListID, score int64, order rank_kit.RankOrder) (float64, error) {
	panic("not implemented") // TODO: Implement
}

func (r *RedisStorage) Register(Score rank_kit.RankScoreIFace) error {
	r.RankScore = Score
	return nil
}

func (r *RedisStorage) SetExpire(expireDay int64) {
	r.ExpiredDay = expireDay
}

func (r *RedisStorage) SetPrefix(prefix string) {
	r.Prefix = prefix
}

func (r *RedisStorage) hashPrefix(rankID rank_kit.RankListID, scope string) string {
	if scope == "" {
		scope = "all"
	}
	return fmt.Sprintf("%s:%s:ctx:%s", r.Prefix, scope, rankID)
}

func (r *RedisStorage) zsetPrefix(rankID rank_kit.RankListID, scope string) string {
	if scope == "" {
		scope = "all"
	}
	return fmt.Sprintf("%s:%s:%s", r.Prefix, scope, rankID)
}

func (r *RedisStorage) itemKey(RankZItemID string) string {
	return RankZItemID
}

// func (r *RedisStorage) SetScope(scope string) error {
// 	r.Scope = scope
// 	return nil
// }

func limitToStop(offset, limit int64) int64 {
	return offset + limit - 1
}

func formatRankItemFromReplyStrings(reply []string) ([]*rank_kit.RankZItem, error) {

	res := make([]*rank_kit.RankZItem, 0, len(reply))

	for i := 0; i < len(reply); i = i + 2 {

		score, err := strconv.Atoi(reply[i+1])
		if err != nil {
			continue
		}

		tmp := &rank_kit.RankZItem{
			ItemContext: rank_kit.ItemContext{
				RankItemID: rank_kit.RankItemID(reply[i]),
			},
			Score: int64(score),
		}

		res = append(res, tmp)
	}

	return res, nil
}

func (r *RedisStorage) multiGetRankItemCtx(ctx context.Context, conn redis.Conn, rankListId rank_kit.RankListID, rankItems []rank_kit.RankItemID, scope string) ([]map[string]string, error) {

	if len(rankItems) == 0 {
		return []map[string]string{}, nil
	}

	ctxParams := r.formatMCtx(rankListId, rankItems, scope)
	ctxArr, err := redis.Strings(conn.Do("HMGET", ctxParams...))
	if err != nil {
		fmt.Println(fmt.Sprintf("multiGetRankItemCtx HMGET err:%s", err.Error()))
		return nil, rank_kit.ErrRankStorageFindRankItem
	}

	var ret []map[string]string
	for _, v := range ctxArr {
		var x = make(map[string]string)
		if err := json.Unmarshal([]byte(v), &x); err != nil {
			fmt.Println(fmt.Sprintf("data err:%s", err.Error()))
		}
		ret = append(ret, x)
	}
	return ret, nil
}

func (r *RedisStorage) formatMCtx(rankListId rank_kit.RankListID, rankItemId []rank_kit.RankItemID, scope string) []interface{} {

	ctxParams := make([]interface{}, 0, len(rankItemId)+1)
	ctxKey := r.hashPrefix(rankListId, scope)
	ctxParams = append(ctxParams, ctxKey)

	for _, item := range rankItemId {
		ctxParams = append(ctxParams, item)
	}

	return ctxParams
}

func JsonMarshal(s map[string]string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func (r *RedisStorage) getRankItemCtx(ctx context.Context, conn redis.Conn, rankListId rank_kit.RankListID, rankItemId rank_kit.RankItemID, scope string) (map[string]string, error) {
	ctxKey := r.hashPrefix(rankListId, scope)
	ctxField := rankItemId

	ctxStr, err := redis.String(conn.Do("HGET", ctxKey, ctxField))
	if err == redis.ErrNil {
		return nil, rank_kit.ErrRankStorageNotFoundItem
	}

	if err != nil {
		fmt.Println(fmt.Sprintf("getRankItemCtx HGET err:%s", err.Error()))
		return nil, err
	}
	var ret map[string]string = make(map[string]string)
	if err := json.Unmarshal([]byte(ctxStr), &ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *RedisStorage) getRawScore(ctx context.Context, rankListId rank_kit.RankListID, id rank_kit.RankItemID, scope string) (int64, error) {
	conn, _ := r.redisPool.GetContext(ctx)
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Println(fmt.Sprintf("redis conn close err:%s", err.Error()))
		}
	}()

	zk := r.zsetPrefix(rankListId, scope)
	zm := id

	score, err := redis.Int64(conn.Do("ZSCORE", zk, zm))

	if err == redis.ErrNil {
		return 0, rank_kit.ErrRankStorageNotFoundItem
	}

	if err != nil {
		return 0, rank_kit.ErrRankStorageFindRankItem
	}

	return score, nil
}
