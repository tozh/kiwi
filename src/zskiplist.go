package src

import "math/rand"

type ZSkiplistNode struct {
	ele string
	score float64
	backward *ZSkiplistNode
	level []ZSkiplistLevel
}

type ZSkiplistLevel struct {
	forward *ZSkiplistNode
	span int
}

type ZSkiplist struct {
	header *ZSkiplistNode
	tail * ZSkiplistNode
	len int
	level int
}

// Struct to hold a inclusive/exclusive range spec by score comparison
type ZScoreRangeSpec struct {
	min float64
	max float64
	minex bool
	maxex bool
}

// Struct to hold an inclusive/exclusive range spec by lexicographic comparison

type ZLexRangeSpec struct {
	min string
	max string
	minex bool
	maxex bool
}

/* we assume the element is not already inside, since we allow duplicated
 * scores, reinserting the same element should never happen since the
 * caller of zslInsert() should test in the hash table if the element is
 * already inside or not. */
/* ZSkiplist methods */
func (zsl *ZSkiplist) ZSkiplistInsert(score float64, ele string) *ZSkiplistNode{
	update := [ZSKIPLIST_MAXLEVEL]*ZSkiplistNode{}
	rank := [ZSKIPLIST_MAXLEVEL]int{}
	x := zsl.header
	// find the position for all level that below the zsl.level -> put into the update and rank
	for i:=zsl.level-1;i>=0;i-- {
		/* store rank that is crossed to reach the insert position */
		if i==zsl.level-1 {
			rank[i] = 0
		} else {
			rank[i] = rank[i+1]
		}
		for x.level[i].forward != nil &&
			(x.level[i].forward.score<score ||
				(x.level[i].forward.score==score && x.level[i].forward.ele != ele)) {
			rank[i] += x.level[i].span
			x = x.level[i].forward
		}
		update[i] = x
	}
	/* we assume the element is not already inside, since we allow duplicated
	* scores, reinserting the same element should never happen since the
	* caller of zslInsert() should test in the hash table if the element is
	* already inside or not. */
	level := ZSkiplistRandomLevel()
	if level > zsl.level {
		for i:=zsl.level;i<level;i++ {
			rank[i] = 0
			update[i] = zsl.header
			update[i].level[i].span = zsl.len
		}
		zsl.level = level
	}
	x = ZSkiplistCreateNode(level, score, ele)
	for i:=0;i<level;i++ {
		x.level[i].forward = update[i].level[i].forward
		update[i].level[i].forward=x

		/* update span covered by update[i] as x is inserted here */
		rankDiff := rank[0] - rank[i]
		x.level[i].span = update[i].level[i].span - rankDiff
		update[i].level[i].span = rankDiff + 1
	}
	/* increment span for untouched levels */
	for i:=level;i<zsl.level;i++ {
		update[i].level[i].span++
	}
	if update[0] == zsl.header {
		x.backward = nil
	} else {
		x.backward = update[0]
	}

	if x.level[0].forward != nil {
		x.level[0].forward.backward = x
	} else {
		zsl.tail = x
	}
	zsl.len++
	return x
}

/* Internal function used by zslDelete, zslDeleteByScore and zslDeleteByRank */
func (zsl *ZSkiplist) zSkiplistDeleteNode(x *ZSkiplistNode, update [ZSKIPLIST_MAXLEVEL]*ZSkiplistNode) {
	for i:=0;i<zsl.level;i++ {
		if update[i].level[i].forward == x {
			update[i].level[i].forward = x.level[i].forward
			update[i].level[i].span += x.level[i].span - 1
		} else {
			update[i].level[i].span -= 1
		}
	}
	if x.level[0].forward != nil {
		x.level[0].forward.backward = x.backward
	} else {
		zsl.tail = x.backward
	}
	for zsl.level > 1 && zsl.header.level[zsl.level-1].forward != nil {
		zsl.level--
	}
	zsl.len--
}

func (zsl *ZSkiplist) ZSkiplistDelete (score float64, ele string) bool {
	x := zsl.header
	update := [ZSKIPLIST_MAXLEVEL]*ZSkiplistNode{}
	for i:=zsl.level-1;i>=0;i-- {
		for x.level[i].forward != nil &&
			(x.level[i].forward.score<score ||
				(x.level[i].forward.score==score && x.level[i].forward.ele != ele)) {
			x = x.level[i].forward
		}
		update[i] = x
	}
	x = x.level[0].forward

	//if found!
	if x!=nil && x.score==score && x.ele==ele {
		zsl.zSkiplistDeleteNode(x, update)
		return true
	}

	// not found!
	return false
}

/* Returns if there is a part of the zset is in range. */
func (zsl *ZSkiplist) ZSkiplistIsInRange(rangeSpec *ZScoreRangeSpec) bool{
	if rangeSpec.min > rangeSpec.max ||
		(rangeSpec.min == rangeSpec.max && (rangeSpec.minex || rangeSpec.maxex)) {
			return false
	}
	x := zsl.tail
	if x==nil || !ZSkiplistValueGteMin(x.score, rangeSpec) {
		return false
	}
	x = zsl.header.level[0].forward
	if x==nil || !ZSkiplistValueLteMax(x.score, rangeSpec) {
		return false
	}
	return true
}

func (zsl *ZSkiplist) ZSkiplistFirstInRange(rangeSpec *ZScoreRangeSpec) *ZSkiplistNode {
	if !zsl.ZSkiplistIsInRange(rangeSpec) {
		return nil
	}

}

/* ZSkiplistNode functions */
func ZSkiplistCreateNode(level int, score float64, ele string) *ZSkiplistNode{
	zn := ZSkiplistNode{
		ele,
		score,
		nil,
		make([]ZSkiplistLevel, level),
	}
	return &zn
}

/* ZSkiplist functions */
func ZSkiplistCreate() *ZSkiplist {
	zsl := ZSkiplist{}
	zsl.level = 1
	zsl.len = 0
	zsl.header = ZSkiplistCreateNode(ZSKIPLIST_MAXLEVEL, 0, "")
	for l:=0;l<ZSKIPLIST_MAXLEVEL;l++ {
		zsl.header.level[l].forward = nil
		zsl.header.level[l].span = 0
	}
	zsl.header.backward = nil
	zsl.tail = nil
	return &zsl
}

func ZSkiplistRandomLevel() int{
	level := 1
	for int(rand.Intn(0xFFFF))<ZSKIPLIST_RANDOM_MAXLEVEL {
		level++
	}
	if level<ZSKIPLIST_MAXLEVEL {
		return level
	} else {
		return ZSKIPLIST_MAXLEVEL
	}
}

// value greater than (exclusive)
func ZSkiplistValueGteMin(value float64, rangeSpec *ZScoreRangeSpec) bool{
	if rangeSpec.minex { // exclude min
		return value > rangeSpec.min
	} else { // include min
		return value >= rangeSpec.min
	}
}

// value less than max (exclusive)
func ZSkiplistValueLteMax(value float64, rangeSpec *ZScoreRangeSpec) bool {
	if rangeSpec.maxex { // exclude max
		return value < rangeSpec.max
	} else { // include max
		return value >= rangeSpec.max
	}
}