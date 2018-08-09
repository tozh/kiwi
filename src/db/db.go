package db

import (
	. "redigo/src/object"
	. "redigo/src/structure"
)

type Db struct {
	Dict map[string]*IObject
	//Expires map[*Object]int
	Id int
	//AvgTTL int
	WatchedKeys map[string] int
	DefragLater *List
}

func (db *Db) getKey(key string) *IObject {
	return db.Dict[key]
}

func (db *Db) getKey(key string) *IObject {
	return db.Dict[key]
}

func (db *Db) setKey(key string, ptr *IObject) {
}

