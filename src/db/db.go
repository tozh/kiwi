package db

import (
	. "redigo/src/object"
)

type Db struct {
	Dict map[string]*IObject
	//Expires map[*Object]int
	Id int
	//AvgTTL int
	//WatchedKeys map[string] int
	//DefragLater *List
}

func (db *Db) get(key string) (string, *IObject) {
	return key, db.Dict[key]
}

func (db *Db) getForWrite(key string) (string, *IObject) {
	return key, db.Dict[key]
}

func (db *Db) randGet(key string) (string, *IObject) {
	for key, value := range db.Dict {
		return key, value
	}
	return "", nil
}

func (db *Db) set(key string, ptr *IObject) {
	db.Dict[key] = ptr
}

func (db *Db) delete(key string) {
	delete(db.Dict, key)
}

func (db *Db) setNx(key string, ptr *IObject) bool{
	if _, value := db.get(key); value != nil {
		return false
	} else {
		db.set(key, ptr)
		return true
	}
}

func (db *Db) exist(key string) bool {
	_, value := db.get(key)
	return value != nil
}

