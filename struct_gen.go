package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)

func generateStructCode(database, table string) {
	db, err := sql.Open("sqlite3", database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(fmt.Sprintf("select * from %s", table))
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	cTypes, err := rows.ColumnTypes()
	if err != nil {
		log.Fatal(err)
	}
	cols := []string{}
	for _, c := range cTypes {
		cols = append(cols, c.Name())
	}

	file, err := os.Create(fmt.Sprintf("%s_entity.go", table))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	file.Write([]byte("package main\n"))
	file.Write([]byte(fmt.Sprintf("type %sEntity struct {\n", table)))
	for _, v := range cols {
		line := fmt.Sprintf("%s\t &interface{}", v)
		file.Write([]byte(line))
	}
}
