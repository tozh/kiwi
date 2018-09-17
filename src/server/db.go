package server

import (
	"sync"
)

type Db struct {
	dict  map[string]Objector
	id    int
	mutex sync.RWMutex
}

func (db *Db) Get(key string) Objector {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	return db.dict[key]
}

//func (db *Db) GetForWrite(key Objector) Objector {
//	return db.dict[key]
//}

func (db *Db) RandGet() (string, Objector) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	for key, value := range db.dict {
		return key, value
	}
	return "", nil
}

func (db *Db) Set(key string, ptr Objector) {
	db.mutex.Lock()
	db.dict[key] = ptr
	db.mutex.Unlock()
}

func (db *Db) Delete(key string) {
	db.mutex.Lock()
	delete(db.dict, key)
	db.mutex.Unlock()
}

func (db *Db) SetNx(key string, ptr Objector) bool {
	if value := db.Get(key); value != nil {
		return false
	} else {
		db.Set(key, ptr)
		return true
	}
}

func (db *Db) SetEx(key string, ptr Objector) bool {
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

func (db *Db) Size() int {
	db.mutex.Lock()
	defer db.mutex.RUnlock()
	return len(db.dict)
}

func (db *Db) FlushAll() {
	db.mutex.Lock()
	db.dict = make(map[string]Objector)
	db.mutex.Unlock()
}

func CreateDb(id int) *Db {
	return &Db{
		make(map[string]Objector),
		id,
		sync.RWMutex{},
	}
}
