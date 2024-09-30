package db

import (
	"fmt"
	"os"
	"reflect"

	tinydb "github.com/Yiwen-Chan/tinydb"
)

type DB struct {
	tdb *tinydb.Database
}

var instance map[string]*DB = make(map[string]*DB)

func Instance(name string) *DB {
	if db, ok := instance[name]; ok {
		return db
	}
	os.MkdirAll("data", 0o700)
	db, err := NewDB("data/hiddify-db-" + name + ".json")
	if err != nil {
		fmt.Println("Default DB instance failed", err)
	}
	instance[name] = db
	return db
}

func NewDB(path string) (*DB, error) {
	storage, err := tinydb.JSONStorage(path)
	if err != nil {
		return nil, err
	}
	tdb, err := tinydb.TinyDB(storage)
	if err != nil {
		return nil, err
	}
	return &DB{
		tdb: tdb,
	}, nil
}

func (d *DB) Close() error {
	return d.tdb.Close()
}

func GetTableDB[T any](db *DB) *Table[T] {
	tt := tinydb.GetTable[T](db.tdb)
	if tt == nil {
		return nil
	}
	return &Table[T]{
		Table: tt,
	}
}

func GetTable[T any]() *Table[T] {
	var t T
	name := reflect.TypeOf(t).Name()

	tt := tinydb.GetTable[T](Instance(name).tdb)
	if tt == nil {
		return nil
	}
	return &Table[T]{
		Table: tt,
	}
}

type Table[T any] struct {
	*tinydb.Table[T]
}

func (tbl *Table[T]) Select(selector func(T) bool) ([]T, error) {
	return tbl.Table.Select(selector)
}

func (tbl *Table[T]) All() ([]T, error) {
	return tbl.Table.Select(func(T) bool {
		return true
	})
}

func (tbl *Table[T]) Insert(items ...T) error {
	return tbl.Table.Insert(items...)
}

func (tbl *Table[T]) Delete(selector func(T) bool) ([]T, error) {
	return tbl.Table.Delete(selector)
}

func (tbl *Table[T]) Update(update func(T) T, selector func(T) bool) error {
	return tbl.Table.Update(update, selector)
}

func (tbl *Table[T]) First(selector func(T) bool) (*T, error) {
	data, err := tbl.Select(selector)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("not found")
	}
	return &data[0], nil
}

func (table *Table[T]) FirstOrInsert(selector func(d T) bool, generator func() T) (*T, error) {
	data, err := table.First(selector)
	if err == nil {
		return data, nil
	}

	if err := table.Insert(generator()); err != nil {
		return nil, err
	}
	return table.First(selector)
}

func (table *Table[T]) ReplaceOrInsert(selector func(d T) bool, generator T) error {
	data, err := table.First(selector)
	if err == nil && data != nil {
		if _, err := table.Delete(selector); err != nil {
			return err
		}
	}
	if err := table.Insert(generator); err != nil {
		return err
	}
	return nil
}
