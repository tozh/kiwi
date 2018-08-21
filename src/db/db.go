package db

import (
	. "redigo/src/object"
)

type Db struct {
	Dict map[*StrObject]IObject
	//Expires map[IObject]int64
	Id int64
	//AvgTTL int64
	//WatchedKeys map[string] int64
	//DefragLater *List
}

func (db *Db) get(key *StrObject) (*StrObject, IObject) {
	return key, db.Dict[key]
}

func (db *Db) getForWrite(key *StrObject) (*StrObject, IObject) {
	return key, db.Dict[key]
}

func (db *Db) randGet(key *StrObject) (*StrObject, IObject) {
	for key, value := range db.Dict {
		return key, value
	}
	return nil, nil
}

func (db *Db) set(key *StrObject, ptr IObject) {
	db.Dict[key] = ptr
}

func (db *Db) delete(key *StrObject) {
	delete(db.Dict, key)
}

func (db *Db) setnx(key *StrObject, ptr IObject) bool{
	if _, value := db.get(key); value != nil {
		return false
	} else {
		db.set(key, ptr)
		return true
	}
}

func (db *Db) setex(key *StrObject, ptr IObject) bool {
	if _, value := db.get(key); value != nil {
		db.set(key, ptr)
		return true
	} else {
		return false
	}
}

func (db *Db) exist(key *StrObject) bool {
	_, value := db.get(key)
	return value != nil
}

func (db *Db) size() int64 {
	return int64(len(db.Dict))
}

func (db *Db) empty() {
	db.Dict = make(map[*StrObject] IObject)
}


