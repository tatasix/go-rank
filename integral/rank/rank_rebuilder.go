package rank

import (
	"context"
	"demo/lock"
	"errors"
	"fmt"
	"sync"

	"demo/rank/rank_kit"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/multierr"
)

type RebuildLockIFace interface {
	Lock(ctx context.Context, key string) (lock.LockInstanceIFace, error)
}

var (
	ErrRankListRebuilding = errors.New("排行榜已经在重建")
)

var _ rank_kit.RankRebuildIFace = (*RankRebuild)(nil)

type RankRebuild struct {
	rank_kit.RankSourceIFace
	rank_kit.RankWriter
	rebuildLock RebuildLockIFace

	workNum int   // 工作者数量
	limit   int64 // 分页步长
	pool    *ants.PoolWithFunc
}

// 读取任务
type readTask struct {
	RankListID rank_kit.RankListID
	ctx        context.Context
	offset     int64
	limit      int64
	wg         *sync.WaitGroup
	nums       chan int
	err        chan<- error
	done       chan struct{}
}

func newReadTask(ctx context.Context, RankListID rank_kit.RankListID, offset int64, limit int64, wg *sync.WaitGroup, nums chan int, err chan<- error, done chan struct{}) *readTask {
	return &readTask{ctx: ctx, RankListID: RankListID, offset: offset, limit: limit, wg: wg, nums: nums, err: err, done: done}
}

func (p *readTask) inputError(err error) {
	select {
	case <-p.done:
	case p.err <- err:
	}
}

func (p *readTask) inputNum(num int) {
	select {
	case <-p.done:
	case p.nums <- num:
	}
}

type TYPE_METADATA_CUSTOM string

const METADATA_CUSTOM TYPE_METADATA_CUSTOM = "custom"

func NewRankRebuild(rankSourceIFace rank_kit.RankSourceIFace, rankWriter rank_kit.RankWriter, workNum int, limit int64, rebuildLock RebuildLockIFace) (*RankRebuild, error) {

	r := &RankRebuild{
		RankSourceIFace: rankSourceIFace,
		RankWriter:      rankWriter,
		rebuildLock:     rebuildLock,
		workNum:         workNum,
		limit:           limit,
	}

	var err error
	r.pool, err = ants.NewPoolWithFunc(r.workNum, r.process)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *RankRebuild) process(input interface{}) {

	readTask, ok := input.(*readTask)
	if !ok {
		return
	}

	defer readTask.wg.Done()

	ris, err := r.RankSourceIFace.RankSourceList(readTask.ctx, readTask.RankListID, readTask.offset, readTask.limit)
	if err != nil {
		readTask.inputError(err)
		return
	}

	// 记录每次写入
	readTask.inputNum(len(ris))

	// 获取年月
	yearMonth := readTask.ctx.Value(METADATA_CUSTOM)
	if yearMonth == "" {
		yearMonth = "all"
	}

	// writer work
	_, err = r.RankWriter.BulkStoreRankItem(readTask.ctx, readTask.RankListID, ris, yearMonth.(string))
	if err != nil {
		readTask.inputError(err)
	}

}

// Rebuild NOTE 这是一种不在知道TOTAL的情况进行数据获取
func (r *RankRebuild) Rebuild(ctx context.Context, RankListID rank_kit.RankListID) (int, error) {
	if locker, err := r.rebuildLock.Lock(ctx, string(RankListID)); err != nil {
		return 0, err
	} else {
		defer func() {
			locker.Release(ctx)
		}()
	}

	// rebuild task start
	var result error
	var num int

	err := make(chan error)
	nums := make(chan int)
	done := make(chan struct{}, 1)
	finish := make(chan struct{}, 1)
	collectorFinish := make(chan struct{}, 1)

	wg := new(sync.WaitGroup)

	// 开启一个协程负责收集反馈的信息
	go func() {

		for {
			select {
			case <-done:
				close(collectorFinish)
				return
			case e := <-err:
				// 发生错误 终止任务
				result = multierr.Append(result, e)
				select {
				case <-finish:
				default:
					close(finish)
				}
			case n := <-nums:
				num = num + n
				// 如果收到0长度，证明已经循环到尾了
				if n == 0 {
					select {
					case <-finish:
					default:
						close(finish)
					}
				}
			}
		}

	}()

	// 循环提交读取数据的任务
ReadLoop:
	for i := 0; ; i++ {
		select {
		case <-finish:
			break ReadLoop
		default:
			wg.Add(1)
			offset := int64(i) * r.limit
			_ = r.pool.Invoke(newReadTask(ctx, RankListID, offset, r.limit, wg, nums, err, done))
		}
	}

	// 等待所有任务处理完成
	wg.Wait()
	close(done)
	<-collectorFinish

	return num, result
}

func (r *RankRebuild) RebuildAll(ctx context.Context) (int, error) {

	var offset, limit int64 = 0, r.limit
	var rows int
	for {

		RankListIDs, err := r.RankSourceIFace.RankSourceRankList(ctx, offset, limit)

		if err != nil {
			fmt.Println(fmt.Sprintf("RankRebuild.RebuildAll RankSourceIFace.RankSourceRankList err:%s", err.Error()))
			break
		}

		if len(RankListIDs) == 0 {
			break
		}

		offset = offset + limit

		for _, RankListID := range RankListIDs {
			if r, err := r.Rebuild(ctx, RankListID); err != nil {
				fmt.Println(fmt.Sprintf("RankRebuild.RebuildAll Rebuild err:%s", err.Error()))
			} else {
				rows = rows + r
			}
		}

	}

	return rows, nil
}
