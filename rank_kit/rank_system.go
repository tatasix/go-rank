package rank_kit

type RankSystemIFace interface {
	RankSourceIFace
	RankScoreIFace
	RankStorageIFace
	RankRebuildIFace
}
