package rank

type IntegralRank struct {
	UserID      uint64
	ListId      uint64
	YearMonth   string
	AddIntegral uint64
	Context     string
	RecordTime  uint32
}

type IntegralChange struct {
	Type            uint32 // 1 添加 2 减少
	UserId          uint64 // 用户信息
	ListId          uint64 // 指定激励账户
	IntegralBalance int64  // 当前账户积分余额
	Integral        int64  // 本次操作积分
	UpdateTime      int64  // 本次操作时间
}

type IntegralRankListRequest struct {
	ListId uint64
	UserId uint64
	Year   uint32
	Month  uint32
	Len    uint32
}

type IntegralRankListResponse struct {
	Self        *RankItem
	List        []*RankItem
	ListId      uint64
	UpdatedTime int64
}

type RankItem struct {
	UserId             int64
	Rank               uint32
	AddIntegralByMonth int64
	Status             uint32
}

type UserInfo struct {
	UserID    uint64 // 用户ID
	UserPhone string // 手机号码
	UserName  string // 用户名称
	Avatar    string // 头像
}
