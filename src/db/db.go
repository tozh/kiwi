package db

import (
	. "redigo/src/object"
	. "redigo/src/constant"
	"strconv"
)

type Db struct {
	Dict map[string]IObject
	//Expires map[IObject]int64
	Id int64
	//AvgTTL int64
	//WatchedKeys map[string] int64
	//DefragLater *List
}

func getStrByStrObject(key *StrObject) string {
	if key.Encoding == OBJ_ENCODING_INT {
		return strconv.FormatInt(*key.Value.(*int64), 10)
	} else {
		return *key.Value.(*string)
	}
}

func (db *Db) Get(key string) IObject {
	return db.Dict[key]
}

//func (db *Db) GetForWrite(key IObject) IObject {
//	return db.Dict[key]
//}

func (db *Db) RandGet() IObject {
	for _, value := range db.Dict {
		return value
	}
	return nil
}

func (db *Db) Set(key string, ptr IObject) {
	db.Dict[key] = ptr
}

func (db *Db) Delete(key string) {
	delete(db.Dict, key)
}

func (db *Db) SetNx(key string, ptr IObject) bool{
	if value := db.Get(key); value != nil {
		return false
	} else {
		db.Set(key, ptr)
		return true
	}
}

func (db *Db) SetEx(key string, ptr IObject) bool {
	if value := db.Get(key); value != nil {
		db.Set(key, ptr)
		return true
	} else {
		return false
	}
}

func (db *Db) Exist(key string) bool {
	value := db.Get(key)
	return value != nil
}

func (db *Db) Size() int64 {
	return int64(len(db.Dict))
}

func (db *Db) FlushAll() {
	db.Dict = make(map[string] IObject)
}



