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

func (db *Db) Get(key *StrObject) IObject {
	return db.Dict[key]
}

func (db *Db) GetForWrite(key *StrObject) IObject {
	return db.Dict[key]
}

func (db *Db) RandGet(key *StrObject) IObject {
	for _, value := range db.Dict {
		return value
	}
	return nil
}

func (db *Db) Set(key *StrObject, ptr IObject) {
	db.Dict[key] = ptr
}

func (db *Db) Delete(key *StrObject) {
	delete(db.Dict, key)
}

func (db *Db) SetNx(key *StrObject, ptr IObject) bool{
	if value := db.Get(key); value != nil {
		return false
	} else {
		db.Set(key, ptr)
		return true
	}
}

func (db *Db) SetEx(key *StrObject, ptr IObject) bool {
	if value := db.Get(key); value != nil {
		db.Set(key, ptr)
		return true
	} else {
		return false
	}
}

func (db *Db) Exist(key *StrObject) bool {
	value := db.Get(key)
	return value != nil
}

func (db *Db) Size() int64 {
	return int64(len(db.Dict))
}

func (db *Db) FlushAll() {
	db.Dict = make(map[*StrObject] IObject)
}


