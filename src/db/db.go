package db

import (
	. "redigo/src/object"
)

type Db struct {
	Dict map[IObject]IObject
	//Expires map[IObject]int64
	Id int64
	//AvgTTL int64
	//WatchedKeys map[string] int64
	//DefragLater *List
}

func (db *Db) Get(key IObject) IObject {
	return db.Dict[key]
}

func (db *Db) GetForWrite(key IObject) IObject {
	return db.Dict[key]
}

func (db *Db) RandGet(key IObject) IObject {
	for _, value := range db.Dict {
		return value
	}
	return nil
}

func (db *Db) Set(key IObject, ptr IObject) {
	db.Dict[key] = ptr
}

func (db *Db) Delete(key IObject) {
	delete(db.Dict, key)
}

func (db *Db) SetNx(key IObject, ptr IObject) bool{
	if value := db.Get(key); value != nil {
		return false
	} else {
		db.Set(key, ptr)
		return true
	}
}

func (db *Db) SetEx(key IObject, ptr IObject) bool {
	if value := db.Get(key); value != nil {
		db.Set(key, ptr)
		return true
	} else {
		return false
	}
}

func (db *Db) Exist(key IObject) bool {
	value := db.Get(key)
	return value != nil
}

func (db *Db) Size() int64 {
	return int64(len(db.Dict))
}

func (db *Db) FlushAll() {
	db.Dict = make(map[IObject] IObject)
}


