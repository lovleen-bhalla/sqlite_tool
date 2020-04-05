package main

import (
	"database/sql"
	"fmt"
	"log"
)

func getTableSchema(database, table string) string {
	db, err := sql.Open("sqlite3", database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	row, err := db.Query(fmt.Sprintf("select sql from sqlite_master where type='table' and name='%s'", table))
	if err != nil {
		log.Fatal(err)
	}
	defer row.Close()

	var statement string
	for row.Next() {
		err = row.Scan(&statement)
	}
	if err != nil {
		log.Fatal(err)
	}
	return statement
}
