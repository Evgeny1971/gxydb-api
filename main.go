// main.go

package main

import "os"

func main() {
	a := App{}
	a.initOidc(os.Getenv("ACC_URL"))
	a.Initialize(
		os.Getenv("APP_DB_USERNAME"),
		os.Getenv("APP_DB_PASSWORD"),
		os.Getenv("APP_DB_NAME"))
	a.Run(":8080")
}

//const createUConvertTable = `CREATE TABLE IF NOT EXISTS rooms
//(
//room INT,
//janus TEXT NOT NULL,
//description TEXT,
//CONSTRAINT room_pkey PRIMARY KEY (room)
//);`