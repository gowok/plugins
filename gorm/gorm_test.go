package gorm

import (
	"testing"

	"gorm.io/gorm"
)

func TestDBNoDefaultWithoutName(t *testing.T) {
	prev := dbs
	dbs = dbMap{"default": &gorm.DB{}}
	t.Cleanup(func() { dbs = prev })

	if _, ok := DBNoDefault().Get(); !ok {
		t.Fatal("DBNoDefault() without name should return the default connection")
	}

	if _, ok := DBNoDefault("missing").Get(); ok {
		t.Fatal(`DBNoDefault("missing") should not fall back to the default connection`)
	}
}
