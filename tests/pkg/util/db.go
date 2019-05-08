package util

import (
	"database/sql"

	"github.com/golang/glog"
)

// OpenDB opens db
func OpenDB(dsn string, maxIdleConns int) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(maxIdleConns)
	glog.V(4).Info("DB opens successfully")
	return db, nil
}
