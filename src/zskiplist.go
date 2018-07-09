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

//type ZLexRangeSpec struct {
//	min string
//	max string
//	minex bool
//	maxex bool
//}

/* we assume the element is not already inside, since we allow duplicated
 * scores, reinserting the same element should never happen since the
 * caller of zslInsert() should test in the hash table if the element is
 * already inside or not. */
/* -------------------------- ZSkiplist methods -------------------------- */
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


/* Find the first node that is contained in the specified range.
 * Returns NULL when no element is contained in the range. */
func (zsl *ZSkiplist) ZSkiplistFirstInRange(rangeSpec *ZScoreRangeSpec) *ZSkiplistNode {
	if !zsl.ZSkiplistIsInRange(rangeSpec) {
		return nil
	}
	x := zsl.header
	for i:=zsl.level-1;i>=0;i-- {
		// go forward until *OUT* of range
		for x.level[i].forward !=nil && !ZSkiplistValueGteMin(x.level[i].forward.score, rangeSpec) {
			x = x.level[i].forward
		}
	}
	x = x.level[0].forward
	// check if score <= max
	if x == nil || !ZSkiplistValueLteMax(x.score, rangeSpec) {
		return nil
	}
	return x
}

/* Find the last node that is contained in the specified range.
 * Returns NULL when no element is contained in the range. */
func (zsl *ZSkiplist) ZSkiplistLastInRange(rangeSpec *ZScoreRangeSpec) *ZSkiplistNode {
	if !zsl.ZSkiplistIsInRange(rangeSpec) {
		return nil
	}
	x := zsl.header
	for i:=zsl.level-1;i>=0;i-- {
		// go forward while *IN* range
		for x.level[i].forward !=nil && ZSkiplistValueLteMax(x.level[i].forward.score, rangeSpec) {
			x = x.level[i].forward
		}
	}
	// check if score >= min
	if x == nil || !ZSkiplistValueGteMin(x.score, rangeSpec) {
		return nil
	}
	return x
}


/* Delete all the elements with score between min and max from the skiplist.
 * Min and max are inclusive, so a score >= min || score <= max is deleted.
 * Note that this function takes the reference to the hash table view of the
 * sorted set, in order to remove the elements from the hash table too. */
func (zsl *ZSkiplist) ZSkiplistDeleteRangeByScore(rangeSpec *ZScoreRangeSpec, dict map[string]*ZSkiplistNode) int{
	update := [ZSKIPLIST_MAXLEVEL]*ZSkiplistNode{}
	x := zsl.header
	removed := 0
	for i:=zsl.level-1;i>=0;i-- {
		lteMin := x.level[i].forward.score < rangeSpec.min
		if !lteMin { // (!lteMin) ==> (score >= min), check if score == min is admitted
			lteMin = rangeSpec.minex && x.level[i].forward.score == rangeSpec.min
		}

		for x.level[i].forward != nil && lteMin {
			x = x.level[i].forward
		}
		update[i] = x
	}
	// current node is the last with score < or <= min

	x = x.level[0].forward

	//delete while in range
	if rangeSpec.maxex { // exclude max
		for x != nil && x.score < rangeSpec.max {
			next := x.level[0].forward
			zsl.zSkiplistDeleteNode(x, update)
			delete(dict, x.ele)
			removed++
			x = next
		}
	} else { // not exclude max
		for x != nil && x.score <= rangeSpec.max {
			next := x.level[0].forward
			zsl.zSkiplistDeleteNode(x, update)
			delete(dict, x.ele)
			removed++
			x = next
		}
	}

	return removed
}

//func (zsl *ZSkiplist) ZSkiplistDeleteRangeByLex(rangeSpec *ZLexRangeSpec, dict map[string]*ZSkiplistNode) int {
//	update := [ZSKIPLIST_MAXLEVEL]*ZSkiplistNode{}
//	x := zsl.header
//	removed := 0
//	for i:=zsl.level-1;i>=0;i-- {
//		for x.level[i].forward != nil && !ZSkiplistLexValueGteMin(x.level[i].forward.ele, rangeSpec) {
//			x = x.level[i].forward
//		}
//		update[i] = x
//	}
//	// current node is the last with ele.value < or <= min
//
//	x = x.level[0].forward
//
//	// delete nodes while in range
//	for x!=nil && ZSkiplistLexValueLteMax(x.ele, rangeSpec) {
//		next := x.level[0].forward
//		zsl.zSkiplistDeleteNode(x, update)
//		delete(dict, x.ele)
//		removed++
//		x = next
//	}
//	return removed
//}


func (zsl *ZSkiplist) ZSkiplistDeleteRangeByRank(start int, end int, dict map[string]*ZSkiplistNode) int {
	update := [ZSKIPLIST_MAXLEVEL]*ZSkiplistNode{}
	traversed := 0
	removed := 0
	x := zsl.header
	for i:=zsl.level-1;i>=0;i-- {
		for x.level[i].forward != nil && traversed+x.level[i].span < start {
			traversed += x.level[i].span
			x = x.level[i].forward
		}
		update[i] = x
	}
	traversed++
	x = x.level[0].forward
	for x != nil && traversed <= end {
		next := x.level[0].forward
		zsl.zSkiplistDeleteNode(x, update)
		delete(dict, x.ele)
		removed++
		x = next
	}
	return removed
}

func (zsl *ZSkiplist) ZSkiplistGetRank(score float64, ele string) int {
	rank := 0
	x := zsl.header
	for i:=zsl.level-1;i>=0;i-- {
		for x.level[i].forward != nil && (x.level[i].forward.score < score ||
			(x.level[i].forward.score == score && x.level[i].forward.ele <= ele)) {
			rank += x.level[i].span
			x = x.level[i].forward
		}
	}
	if x.ele != "" && x.ele <= ele {
		return rank
	}
	return 0
}

func (zsl *ZSkiplist) ZSkiplistGetElementByRank(rank int) *ZSkiplistNode{
	if rank < 0 {
		return nil
	}
	traversed := 0
	x := zsl.header
	for i:=zsl.level-1;i>=0;i-- {
		for x.level[i].forward != nil && (traversed +x.level[i].span) <= rank {
			traversed += x.level[i].span
			x = x.level[i].forward
		}
		if traversed == rank {
			return x
		}
	}
	return nil
}











/* -------------------------- ZSkiplistNode functions -------------------------- */
func ZSkiplistCreateNode(level int, score float64, ele string) *ZSkiplistNode{
	zn := ZSkiplistNode{
		ele,
		score,
		nil,
		make([]ZSkiplistLevel, level),
	}
	return &zn
}

/* -------------------------- ZSkiplist functions -------------------------- */
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

// value greater than (exclusively) min
func ZSkiplistValueGteMin(value float64, rangeSpec *ZScoreRangeSpec) bool{
	if rangeSpec.minex { // exclude min
		return value > rangeSpec.min
	} else { // include min
		return value >= rangeSpec.min
	}
}

// value less than max (exclusively) max
func ZSkiplistValueLteMax(value float64, rangeSpec *ZScoreRangeSpec) bool {
	if rangeSpec.maxex { // exclude max
		return value < rangeSpec.max
	} else { // include max
		return value >= rangeSpec.max
	}
}

//func ZSkiplistLexValueGteMin(value string, rangeSpec *ZLexRangeSpec) bool{
//	if rangeSpec.minex {
//		return value > rangeSpec.min
//	} else {
//		return value >= rangeSpec.min
//	}
//}
//
//func ZSkiplistLexValueLteMax(value string, rangeSpec *ZLexRangeSpec) bool{
//	if rangeSpec.maxex {
//		return value < rangeSpec.max
//	} else {
//		return value <= rangeSpec.max
//	}
//}