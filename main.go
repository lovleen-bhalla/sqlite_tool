package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"

	_ "github.com/mattn/go-sqlite3"
)

const dump_usage = `run dump --file={path/to/sqlite/database/file} --table={table name}`
const edit_usage = `run edit --file={path/to/sqlite/database/file} --table={table name} --json:{file containing json to save}`

func main() {
	dump := flag.NewFlagSet("dump", flag.ExitOnError)
	dbfile := dump.String("file", "", dump_usage)
	table := dump.String("table", "", dump_usage)

	edit := flag.NewFlagSet("edit", flag.ExitOnError)
	edit_db := edit.String("file", "", edit_usage)
	edit_table := edit.String("table", "", edit_usage)
	edit_json := edit.String("json", "", edit_usage)

	if len(os.Args) < 2 {
		fmt.Println("Use command dump or save")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "dump":
		dump.Parse(os.Args[2:])
		if *dbfile == "" {
			log.Println("Database name not given")
			log.Println("usage:", dump_usage)
		} else if *table == "" {
			log.Println("Table name not given")
			log.Println("usage:", dump_usage)
		} else {
			saveAndDumpDb(*dbfile, *table)
		}

	case "edit":
		edit.Parse(os.Args[2:])
		if *edit_db == "" {
			log.Println("Database name not given")
			log.Println("usage:", edit_usage)
		} else if *edit_table == "" {
			log.Println("Table name not given")
			log.Println("usage:", edit_usage)
		} else if *edit_json == "" {
			log.Println("File containig json data to dump not given")
			log.Println("usage:", edit_usage)
		} else {
			saveJsonToDb(*edit_db, *edit_table, *edit_json)
		}
	}
}

func saveJsonToDb(dbpath, tableName, jsonFile string) {
	db, err := sql.Open("sqlite3", copyDatabase(dbpath))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("delete from %s", tableName))
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadFile(jsonFile)
	var table []map[string]interface{}
	err = json.Unmarshal(data, &table)
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range table {
		var vals []interface{}
		cols := "( "
		values := "( "
		for key, val := range row {
			cols = cols + key
			values = values + "?"

			if reflect.ValueOf(val).Kind() == reflect.Map {
				var blob interface{}
				blob, err := json.Marshal(val)
				if err != nil {
					log.Fatal(err)
				}
				vals = append(vals, blob)

			} else {
				vals = append(vals, val)
			}
			if len(vals) != len(row) {
				cols = cols + ", "
				values = values + ", "
			}
		}
		cols = cols + " )"
		values = values + ") "
		query := fmt.Sprintf("INSERT INTO %s  %s VALUES %s", tableName, cols, values)
		_, err = db.Exec(query, vals...)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func copyDatabase(database string) string {
	err := os.RemoveAll("generated/")
	if err != nil {
		log.Println(err)
	}
	err = os.MkdirAll("generated/", os.ModeDir|os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	path := path.Join("generated", path.Base(database))
	newFile, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer newFile.Close()

	data, err := ioutil.ReadFile(database)
	if err != nil {
		log.Fatal(err)
	}

	_, err = newFile.Write(data)
	if err != nil {
		log.Fatal(err)
	}
	return path
}

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

func saveAndDumpDb(dbfile, tableName string) {
	db, err := sql.Open("sqlite3", dbfile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(fmt.Sprintf("select * from %s", tableName))
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		log.Fatal(err)
	}

	var table [][]interface{}
	for rows.Next() {
		var row []interface{}
		for i := 0; i < len(colTypes); i++ {
			var a interface{}
			row = append(row, &a)
		}
		err = rows.Scan(row...)
		if err != nil {
			log.Fatal(err)
		}

		table = append(table, row)
	}
	var tm []map[string]interface{}
	for _, k := range table {
		rm := make(map[string]interface{})
		for j, v := range k {
			rValue := reflect.ValueOf(v)
			uv := reflect.Indirect(rValue)
			e := uv.Elem()
			key := colTypes[j].Name()
			//rm[key] = e
			if !e.IsValid() {
				col := colTypes[j]
				switch col.DatabaseTypeName() {
				case "INTEGER":
					rm[key] = 0
					break
				case "BLOB":
					rm[key] = ""
					break
				case "REAL":
					rm[key] = 0
					break
				case "TEXT":
					rm[key] = ""
					break
				}
			} else {
				//	fmt.Println(colTypes[j], e.Type())
				switch e.Kind() {
				case reflect.String:
					var s string = e.String()
					rm[key] = s
					fmt.Sprintln(s)
					break
				case reflect.Int64:
					var i64 int64 = e.Int()
					rm[key] = i64
					break
				case reflect.Float64:
					var f64 float64 = e.Float()
					rm[key] = f64
					break
				case reflect.Slice:
					var bytes []byte
					bytes = e.Bytes()
					var ji interface{}
					jerr := json.Unmarshal(bytes, &ji)
					if jerr != nil {
						log.Println(jerr.Error())
					} else {
						rm[key] = ji
					}
				}
			}

			tm = append(tm, rm)
		}
	}
	js, err := json.MarshalIndent(tm, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	fileName := tableName + ".json"
	err = os.Remove(fileName)
	if err != nil {
		fmt.Println(err)
	}
	err = ioutil.WriteFile(fileName, js, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
}

func printStringSlice(slice []string) {
	for i, val := range slice {
		fmt.Println(i, val)
	}
}

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
