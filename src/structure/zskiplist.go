package structure

import "math/rand"
import ."redigo/src/constant"
type ZSkiplistNode struct {
	Ele      string
	Score    float64
	Backward *ZSkiplistNode
	Level    []ZSkiplistLevel
}

type ZSkiplistLevel struct {
	Forward *ZSkiplistNode
	Span    int
}

type ZSkiplist struct {
	Header *ZSkiplistNode
	Tail   *ZSkiplistNode
	Len    int
	Level  int
}

// Struct to hold a inclusive/exclusive range spec by Score comparison
type ZScoreRangeSpec struct {
	Min   float64
	Max   float64
	Minex bool
	Maxex bool
}

// Struct to hold an inclusive/exclusive range spec by lexicographic comparison

//type ZLexRangeSpec struct {
//	Min string
//	Max string
//	Minex bool
//	Maxex bool
//}

/* we assume the element is not already inside, since we allow duplicated
 * scores, reinserting the same element should never happen since the
 * caller of zslInsert() should test in the hash table if the element is
 * already inside or not. */
/* -------------------------- ZSkiplist methods -------------------------- */
func (zsl *ZSkiplist) ZSkiplistInsert(score float64, ele string) *ZSkiplistNode{
	update := [ZSKIPLIST_MAXLEVEL]*ZSkiplistNode{}
	rank := [ZSKIPLIST_MAXLEVEL]int{}
	x := zsl.Header
	// find the position for all Level that below the zsl.Level -> put into the update and rank
	for i:=zsl.Level -1;i>=0;i-- {
		/* store rank that is crossed to reach the insert position */
		if i==zsl.Level-1 {
			rank[i] = 0
		} else {
			rank[i] = rank[i+1]
		}
		for x.Level[i].Forward != nil &&
			(x.Level[i].Forward.Score <score ||
				(x.Level[i].Forward.Score ==score && x.Level[i].Forward.Ele != ele)) {
			rank[i] += x.Level[i].Span
			x = x.Level[i].Forward
		}
		update[i] = x
	}
	/* we assume the element is not already inside, since we allow duplicated
	* scores, reinserting the same element should never happen since the
	* caller of zslInsert() should test in the hash table if the element is
	* already inside or not. */
	level := ZSkiplistRandomLevel()
	if level > zsl.Level {
		for i:=zsl.Level;i<level;i++ {
			rank[i] = 0
			update[i] = zsl.Header
			update[i].Level[i].Span = zsl.Len
		}
		zsl.Level = level
	}
	x = ZSkiplistCreateNode(level, score, ele)
	for i:=0;i<level;i++ {
		x.Level[i].Forward = update[i].Level[i].Forward
		update[i].Level[i].Forward =x

		/* update Span covered by update[i] as x is inserted here */
		rankDiff := rank[0] - rank[i]
		x.Level[i].Span = update[i].Level[i].Span - rankDiff
		update[i].Level[i].Span = rankDiff + 1
	}
	/* increment Span for untouched levels */
	for i:=level;i<zsl.Level;i++ {
		update[i].Level[i].Span++
	}
	if update[0] == zsl.Header {
		x.Backward = nil
	} else {
		x.Backward = update[0]
	}

	if x.Level[0].Forward != nil {
		x.Level[0].Forward.Backward = x
	} else {
		zsl.Tail = x
	}
	zsl.Len++
	return x
}

/* Internal function used by zslDelete, zslDeleteByScore and zslDeleteByRank */
func (zsl *ZSkiplist) ZSkiplistDeleteNode(x *ZSkiplistNode, update [ZSKIPLIST_MAXLEVEL]*ZSkiplistNode) {
	for i:=0;i<zsl.Level;i++ {
		if update[i].Level[i].Forward == x {
			update[i].Level[i].Forward = x.Level[i].Forward
			update[i].Level[i].Span += x.Level[i].Span - 1
		} else {
			update[i].Level[i].Span -= 1
		}
	}
	if x.Level[0].Forward != nil {
		x.Level[0].Forward.Backward = x.Backward
	} else {
		zsl.Tail = x.Backward
	}
	for zsl.Level > 1 && zsl.Header.Level[zsl.Level-1].Forward != nil {
		zsl.Level--
	}
	zsl.Len--
}

func (zsl *ZSkiplist) ZSkiplistDelete (score float64, ele string) bool {
	x := zsl.Header
	update := [ZSKIPLIST_MAXLEVEL]*ZSkiplistNode{}
	for i:=zsl.Level -1;i>=0;i-- {
		for x.Level[i].Forward != nil &&
			(x.Level[i].Forward.Score <score ||
				(x.Level[i].Forward.Score ==score && x.Level[i].Forward.Ele != ele)) {
			x = x.Level[i].Forward
		}
		update[i] = x
	}
	x = x.Level[0].Forward

	//if found!
	if x!=nil && x.Score ==score && x.Ele ==ele {
		zsl.ZSkiplistDeleteNode(x, update)
		return true
	}

	// not found!
	return false
}

/* Returns if there is a part of the zset is in range. */
func (zsl *ZSkiplist) ZSkiplistIsInRange(rangeSpec *ZScoreRangeSpec) bool{
	if rangeSpec.Min > rangeSpec.Max ||
		(rangeSpec.Min == rangeSpec.Max && (rangeSpec.Minex || rangeSpec.Maxex)) {
			return false
	}
	x := zsl.Tail
	if x==nil || !ZSkiplistValueGteMin(x.Score, rangeSpec) {
		return false
	}
	x = zsl.Header.Level[0].Forward
	if x==nil || !ZSkiplistValueLteMax(x.Score, rangeSpec) {
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
	x := zsl.Header
	for i:=zsl.Level -1;i>=0;i-- {
		// go Forward until *OUT* of range
		for x.Level[i].Forward !=nil && !ZSkiplistValueGteMin(x.Level[i].Forward.Score, rangeSpec) {
			x = x.Level[i].Forward
		}
	}
	x = x.Level[0].Forward
	// check if Score <= Max
	if x == nil || !ZSkiplistValueLteMax(x.Score, rangeSpec) {
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
	x := zsl.Header
	for i:=zsl.Level -1;i>=0;i-- {
		// go Forward while *IN* range
		for x.Level[i].Forward !=nil && ZSkiplistValueLteMax(x.Level[i].Forward.Score, rangeSpec) {
			x = x.Level[i].Forward
		}
	}
	// check if Score >= Min
	if x == nil || !ZSkiplistValueGteMin(x.Score, rangeSpec) {
		return nil
	}
	return x
}


/* Delete all the elements with Score between Min and Max from the skiplist.
 * Min and Max are inclusive, so a Score >= Min || Score <= Max is deleted.
 * Note that this function takes the reference to the hash table view of the
 * sorted set, in order to remove the elements from the hash table too. */
func (zsl *ZSkiplist) ZSkiplistDeleteRangeByScore(rangeSpec *ZScoreRangeSpec, dict map[string]*ZSkiplistNode) int{
	update := [ZSKIPLIST_MAXLEVEL]*ZSkiplistNode{}
	x := zsl.Header
	removed := 0
	for i:=zsl.Level -1;i>=0;i-- {
		lteMin := x.Level[i].Forward.Score < rangeSpec.Min
		if !lteMin { // (!lteMin) ==> (Score >= Min), check if Score == Min is admitted
			lteMin = rangeSpec.Minex && x.Level[i].Forward.Score == rangeSpec.Min
		}

		for x.Level[i].Forward != nil && lteMin {
			x = x.Level[i].Forward
		}
		update[i] = x
	}
	// current node is the last with Score < or <= Min

	x = x.Level[0].Forward

	//delete while in range
	if rangeSpec.Maxex { // exclude Max
		for x != nil && x.Score < rangeSpec.Max {
			next := x.Level[0].Forward
			zsl.ZSkiplistDeleteNode(x, update)
			delete(dict, x.Ele)
			removed++
			x = next
		}
	} else { // not exclude Max
		for x != nil && x.Score <= rangeSpec.Max {
			next := x.Level[0].Forward
			zsl.ZSkiplistDeleteNode(x, update)
			delete(dict, x.Ele)
			removed++
			x = next
		}
	}

	return removed
}

//func (zsl *ZSkiplist) ZSkiplistDeleteRangeByLex(rangeSpec *ZLexRangeSpec, dict map[string]*ZSkiplistNode) int {
//	update := [ZSKIPLIST_MAXLEVEL]*ZSkiplistNode{}
//	x := zsl.Header
//	removed := 0
//	for i:=zsl.Level-1;i>=0;i-- {
//		for x.Level[i].Forward != nil && !ZSkiplistLexValueGteMin(x.Level[i].Forward.Ele, rangeSpec) {
//			x = x.Level[i].Forward
//		}
//		update[i] = x
//	}
//	// current node is the last with Ele.Value < or <= Min
//
//	x = x.Level[0].Forward
//
//	// delete nodes while in range
//	for x!=nil && ZSkiplistLexValueLteMax(x.Ele, rangeSpec) {
//		Next := x.Level[0].Forward
//		zsl.ZSkiplistDeleteNode(x, update)
//		delete(dict, x.Ele)
//		removed++
//		x = Next
//	}
//	return removed
//}


func (zsl *ZSkiplist) ZSkiplistDeleteRangeByRank(start int, end int, dict map[string]*ZSkiplistNode) int {
	update := [ZSKIPLIST_MAXLEVEL]*ZSkiplistNode{}
	traversed := 0
	removed := 0
	x := zsl.Header
	for i:=zsl.Level -1;i>=0;i-- {
		for x.Level[i].Forward != nil && traversed+x.Level[i].Span < start {
			traversed += x.Level[i].Span
			x = x.Level[i].Forward
		}
		update[i] = x
	}
	traversed++
	x = x.Level[0].Forward
	for x != nil && traversed <= end {
		next := x.Level[0].Forward
		zsl.ZSkiplistDeleteNode(x, update)
		delete(dict, x.Ele)
		removed++
		x = next
	}
	return removed
}

func (zsl *ZSkiplist) ZSkiplistGetRank(score float64, ele string) int {
	rank := 0
	x := zsl.Header
	for i:=zsl.Level -1;i>=0;i-- {
		for x.Level[i].Forward != nil && (x.Level[i].Forward.Score < score ||
			(x.Level[i].Forward.Score == score && x.Level[i].Forward.Ele <= ele)) {
			rank += x.Level[i].Span
			x = x.Level[i].Forward
		}
	}
	if x.Ele != "" && x.Ele <= ele {
		return rank
	}
	return 0
}

func (zsl *ZSkiplist) ZSkiplistGetElementByRank(rank int) *ZSkiplistNode{
	if rank < 0 {
		return nil
	}
	traversed := 0
	x := zsl.Header
	for i:=zsl.Level -1;i>=0;i-- {
		for x.Level[i].Forward != nil && (traversed +x.Level[i].Span) <= rank {
			traversed += x.Level[i].Span
			x = x.Level[i].Forward
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
	zsl.Level = 1
	zsl.Len = 0
	zsl.Header = ZSkiplistCreateNode(ZSKIPLIST_MAXLEVEL, 0, "")
	for l:=0;l<ZSKIPLIST_MAXLEVEL;l++ {
		zsl.Header.Level[l].Forward = nil
		zsl.Header.Level[l].Span = 0
	}
	zsl.Header.Backward = nil
	zsl.Tail = nil
	return &zsl
}

func ZSkiplistRandomLevel() int{
	level := 1
	for float64(rand.Intn(0xFFFF)) < ZSKIPLIST_RANDOM_MAXLEVEL {
		level++
	}
	if level<ZSKIPLIST_MAXLEVEL {
		return level
	} else {
		return ZSKIPLIST_MAXLEVEL
	}
}

// Value greater than (exclusively) Min
func ZSkiplistValueGteMin(value float64, rangeSpec *ZScoreRangeSpec) bool{
	if rangeSpec.Minex { // exclude Min
		return value > rangeSpec.Min
	} else { // include Min
		return value >= rangeSpec.Min
	}
}

// Value less than Max (exclusively) Max
func ZSkiplistValueLteMax(value float64, rangeSpec *ZScoreRangeSpec) bool {
	if rangeSpec.Maxex { // exclude Max
		return value < rangeSpec.Max
	} else { // include Max
		return value >= rangeSpec.Max
	}
}

//func ZSkiplistLexValueGteMin(Value string, rangeSpec *ZLexRangeSpec) bool{
//	if rangeSpec.Minex {
//		return Value > rangeSpec.Min
//	} else {
//		return Value >= rangeSpec.Min
//	}
//}
//
//func ZSkiplistLexValueLteMax(Value string, rangeSpec *ZLexRangeSpec) bool{
//	if rangeSpec.Maxex {
//		return Value < rangeSpec.Max
//	} else {
//		return Value <= rangeSpec.Max
//	}
//}