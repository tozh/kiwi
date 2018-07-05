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




/* ZSkiplist methods */
func (zsl *ZSkiplist) ZSkiplistInsert(score float64, ele string) {
	update := [ZSKIPLIST_MAXLEVEL]ZSkiplistNode{}
	rank := [ZSKIPLIST_MAXLEVEL]int{}
	x := zsl.header
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
		update[i] = *x
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