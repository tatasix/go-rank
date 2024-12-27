package rank

import (
	"context"
	"demo/rank/rank_kit"
	"fmt"
	"golang.org/x/sync/singleflight"
	"strconv"
	"time"
)

type IntegralRankService struct {
	RankSystem rank_kit.RankSystemIFace
}

var singleGroup *singleflight.Group

func NewIntegralRankService() *IntegralRankService {
	integralRankSystem, _ := NewRankSystem()
	return &IntegralRankService{
		RankSystem: integralRankSystem,
	}
}

const MAX_RANK = 100

func (s *IntegralRankService) IntegralChangeHandle(ctx context.Context, opt *IntegralChange) error {

	updateTime := time.Unix(opt.UpdateTime, 0)

	yearMonth := getYearMonth(uint32(updateTime.Year()), uint32(updateTime.Month()))

	item, err := s.RankSystem.GetRankItemById(ctx, rank_kit.RankListID(strconv.Itoa(int(opt.ListId))), rank_kit.RankItemID(strconv.Itoa(int(opt.UserId))), yearMonth)
	if err != nil {

		// 未找到排行榜
		if err == rank_kit.ErrRankStorageNotFoundItem {
			//从db 获取数据，重建
			item = &rank_kit.RankZItem{
				ItemContext: rank_kit.ItemContext{
					RankItemID: rank_kit.RankItemID(strconv.Itoa(int(opt.UserId))),
					Context: map[string]string{
						"业务key": "业务数据",
					},
				},
				Score: 0,
				Time:  time.Now().Unix(),
			}
		} else {
			return err
		}
	}

	item.Score = item.Score + opt.Integral

	// 写入当前配置
	if err = s.RankSystem.StoreRankItem(ctx, rank_kit.RankListID(strconv.Itoa(int(opt.ListId))), item, yearMonth); err != nil {
		return err
	}

	return nil
}

// IntegralRankMonthList 积分排行榜月
func (s *IntegralRankService) IntegralRankMonthList(ctx context.Context, req *IntegralRankListRequest) (*IntegralRankListResponse, error) {
	var updateTime = time.Now().Unix()

	var listId uint64 = req.ListId
	var err error

	yearMonth := getYearMonth(req.Year, req.Month)

	updateTime, isCurrmonth := isCurrentMonth(req.Year, req.Month)

	// 优先读取缓存中的数据
	key := "rank_key"
	itemsI, err, _ := singleGroup.Do(key, func() (interface{}, error) {
		if isCurrmonth {
			// 当月数据直接返回redis实时数据
			return s.RankSystem.RankList(ctx, rank_kit.RankListID(strconv.Itoa(int(listId))), rank_kit.DescRankOrder, 0, MAX_RANK, yearMonth)
		}

		// 数据库查询失败，走redis并重建, 重建会写入缓存配置
		if err := s.Flush(ctx, yearMonth, true, listId); err != nil {
			return nil, err
		}

		items, err := s.RankSystem.RankList(ctx, rank_kit.RankListID(strconv.Itoa(int(listId))), rank_kit.DescRankOrder, 0, MAX_RANK, yearMonth)
		if err != nil {
			return nil, err
		}
		return items, nil

	})
	if err != nil {
		return nil, err
	}

	items := itemsI.([]*rank_kit.RankZItem)
	var list []*RankItem
	var rank = 1

	for _, v := range items {

		userID, err := strconv.Atoi(string(v.ItemContext.RankItemID))
		if err != nil {
			return nil, err
		}

		list = append(list, &RankItem{
			UserId:             int64(userID),
			AddIntegralByMonth: v.Score,
			Rank:               uint32(rank),
			Status:             1,
		})
		rank = rank + 1
	}

	var self = &RankItem{
		UserId: int64(req.UserId),
	}

	zitem, err := s.RankSystem.GetRankItemById(ctx, rank_kit.RankListID(strconv.Itoa(int(listId))), rank_kit.RankItemID(strconv.Itoa(int(req.UserId))), yearMonth)
	if err == nil {
		self.AddIntegralByMonth = zitem.Score
		self.Status = 1
	}

	// 积分相同的排序规则 【 1 2 2 4 5 6 】
	var rankSort = func(list []*RankItem, self *RankItem) []*RankItem {
		var rank uint32 = 1
		for _, v := range list {
			v.Rank = rank
			rank = rank + 1
		}

		var lastScore int64 = 0
		for k, v := range list {
			if lastScore == v.AddIntegralByMonth {
				v.Rank = list[k-1].Rank
			}
			if uint32(self.AddIntegralByMonth) == uint32(v.AddIntegralByMonth) {
				self.Rank = v.Rank
			}
			lastScore = v.AddIntegralByMonth
		}
		return list
	}

	list = rankSort(list, self)

	// 只返回固定长度
	if len(list) > int(req.Len)-1 {
		list = list[0:req.Len]
	}

	return &IntegralRankListResponse{
		Self:        self,
		List:        list,
		ListId:      listId,
		UpdatedTime: updateTime,
	}, err
}

// ListUserInfoByIds 查询用户信息，带缓存
func (s *IntegralRankService) ListUserInfoByIds(ctx context.Context, userID []uint64) (map[uint64]*UserInfo, error) {

	return nil, nil
}

func (s *IntegralRankService) FlushAll(ctx context.Context, yearMonth string, force bool) error {

	// 统计年月的， 利用ctx传递年月信息
	newCtx := context.WithValue(ctx, METADATA_CUSTOM, yearMonth)
	if force {
		succ, err := s.RankSystem.RebuildAll(newCtx)
		if err != nil {
			return err
		}
		fmt.Println(fmt.Sprintf("IntegralRankService, FlushAll succ:%v", succ))

	}

	incentiveIds, err := s.RankSystem.RankSourceRankList(newCtx, 0, 0)
	if err != nil {
		return err
	}

	var flushToRedisMap map[uint64][]*rank_kit.RankZItem = make(map[uint64][]*rank_kit.RankZItem)
	for _, v := range incentiveIds {
		zItems, err := s.RankSystem.RankList(ctx, v, rank_kit.DescRankOrder, 0, 100, yearMonth)
		if err != nil {
			return err
		}

		listId, err := strconv.Atoi(string(v))
		if err != nil {
			return err
		}

		if err := FlushToDB(ctx, uint64(listId), yearMonth, zItems); err != nil {
			return err
		}
		flushToRedisMap[uint64(listId)] = zItems
	}

	// 写入缓存数据
	//for k, v := range flushToRedisMap {
	//	//写入缓存
	//	fmt.Println(k)
	//	fmt.Println(v)
	//	//if err := s.Cache.SetIntegralRankList(ctx, k, yearMonth, v); err != nil {
	//	//	return err
	//	//}
	//}

	return nil
}

func (s *IntegralRankService) Flush(ctx context.Context, yearMonth string, force bool, listId uint64) error {

	// 统计年月的， 利用ctx传递年月信息
	newCtx := context.WithValue(ctx, METADATA_CUSTOM, yearMonth)
	if force {
		succ, err := s.RankSystem.Rebuild(newCtx, rank_kit.RankListID(strconv.Itoa(int(listId))))
		if err != nil {
			return err
		}
		fmt.Println(fmt.Sprintf("IntegralRankService, Flush succ:%v", succ))
	}

	// 从redis读取 刷入db
	zItems, err := s.RankSystem.RankList(ctx, rank_kit.RankListID(strconv.Itoa(int(listId))), rank_kit.DescRankOrder, 0, 100, yearMonth)
	if err != nil {
		return err
	}

	if err := FlushToDB(ctx, listId, yearMonth, zItems); err != nil {
		return err
	}

	// 写入缓存数据

	return nil
}

func FlushToDB(ctx context.Context, listId uint64, yearMonth string, zItems []*rank_kit.RankZItem) error {

	for _, v := range zItems {
		integralRank, err := IntegralRankFromZItem(yearMonth, listId, v)
		if err != nil {
			return err
		}
		//写入db
		fmt.Println(integralRank)

	}
	return nil
}

func getYearMonth(year, month uint32) string {
	return fmt.Sprintf("%v-%v", year, month)
}

var isCurrentMonth = func(year, month uint32) (int64, bool) {
	var updateTime int64
	var isCurrmonth bool = false
	nowYear := time.Now().Year()
	nowMonth := time.Now().Month()
	yearMonth := getYearMonth(year, month)
	if yearMonth == getYearMonth(uint32(nowYear), uint32(nowMonth)) {
		isCurrmonth = true
		updateTime = time.Now().Unix()
	} else {
		updateTime = time.Date(int(year), time.Month(month), 1, 23, 59, 59, 0, time.Local).AddDate(0, 1, -1).Unix()
	}
	return updateTime, isCurrmonth
}
