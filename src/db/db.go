package db

import (
	. "redigo/src/object"
)

type Db struct {
	Dict map[string]IObject
	//Expires map[IObject]int64
	Id int64
	//AvgTTL int64
	//WatchedKeys map[string] int64
	//DefragLater *List
}

func (db *Db) get(key string) (string, IObject) {
	return key, db.Dict[key]
}

func (db *Db) getForWrite(key string) (string, IObject) {
	return key, db.Dict[key]
}

func (db *Db) randGet(key string) (string, IObject) {
	for key, value := range db.Dict {
		return key, value
	}
	return "", nil
}

func (db *Db) set(key string, ptr IObject) {
	db.Dict[key] = ptr
}

func (db *Db) delete(key string) {
	delete(db.Dict, key)
}

func (db *Db) setnx(key string, ptr IObject) bool{
	if _, value := db.get(key); value != nil {
		return false
	} else {
		db.set(key, ptr)
		return true
	}
}

func (db *Db) setex(key string, ptr IObject) bool {
	if _, value := db.get(key); value != nil {
		db.set(key, ptr)
		return true
	} else {
		return false
	}
}

func (db *Db) exist(key string) bool {
	_, value := db.get(key)
	return value != nil
}

func (db *Db) size() int64 {
	return int64(len(db.Dict))
}

func (db *Db) empty() {
	db.Dict = make(map[string] IObject)
}


