package util

import (
	"database/sql"

	"github.com/golang/glog"
)

// MustExec must execute sql or fatal
func MustExec(db *sql.DB, query string, args ...interface{}) sql.Result {
	r, err := db.Exec(query, args...)
	if err != nil {
		glog.Fatalf("exec %s err %v", query, err)
	}
	return r
}

// OpenDB opens db
func OpenDB(dsn string, maxIdleConns int) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(maxIdleConns)
	glog.Info("DB opens successfully")
	return db, nil
}
