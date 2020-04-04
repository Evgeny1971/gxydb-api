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

//const createArchiveTable = `CREATE TABLE IF NOT EXISTS users
//(
//id TEXT NOT NULL,
//display TEXT NOT NULL,
//email TEXT DEFAULT NULL,
//"group" TEXT DEFAULT NULL,
//ip TEXT NOT NULL,
//janus TEXT NOT NULL,
//name TEXT NOT NULL,
//role TEXT NOT NULL,
//system TEXT NOT NULL,
//username TEXT NOT NULL,
//room INT NOT NULL,
//timestamp timestamp NOT NULL DEFAULT now(),
//session BIGINT NOT NULL,
//handle BIGINT NOT NULL,
//rfid BIGINT NOT NULL,
//camera BOOL NOT NULL DEFAULT false,
//question BOOL NOT NULL DEFAULT false,
//self_test BOOL NOT NULL DEFAULT false,
//sound_test BOOL NOT NULL DEFAULT false,
//CONSTRAINT users_pkey PRIMARY KEY (id)
//);`
