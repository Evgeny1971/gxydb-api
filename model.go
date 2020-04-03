// model.go

package main

import (
	"database/sql"
	"encoding/json"
)

type rooms struct {
	Room		int                    `json:"room"`
	Janus		string                 `json:"janus"`
	Questions	bool              	   `json:"questions"`
	Description	string                 `json:"description"`
	Num			int                    `json:"num_users"`
	Users		map[string]interface{} `json:"users"`
}

func getRooms(db *sql.DB) ([]rooms, error) {
	rows, err := db.Query("SELECT * FROM rooms ORDER BY description")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	ar := []rooms{}

	for rows.Next() {
		var i rooms
		if err := rows.Scan(&i.Room, &i.Janus, &i.Description, &i.Num); err != nil {
			return nil, err
		}
		ar = append(ar, i)
	}

	return ar, nil
}

func getGroups(db *sql.DB) ([]rooms, error) {
	rows, err := db.Query("SELECT * FROM rooms ORDER BY description")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	ar := []rooms{}

	for rows.Next() {
		var i rooms
		var obj []byte
		if err := rows.Scan(&i.Room, &i.Janus, &i.Questions, &i.Description, &i.Num, &obj); err != nil {
			return nil, err
		}
		json.Unmarshal(obj, &i.Users)
		ar = append(ar, i)
	}

	return ar, nil
}

func (i *rooms) getRoom(db *sql.DB) error {
	var obj []byte

	err := db.QueryRow("SELECT * FROM rooms WHERE room = $1",
		i.Room).Scan(&i.Room, &i.Janus, &i.Questions, &i.Description, &i.Num, &obj)

	if err != nil {
		return err
	}
	err = json.Unmarshal(obj, &i.Users)

	return err
}

func (i *rooms) postRoom(db *sql.DB) error {

	err := db.QueryRow(
		"INSERT INTO rooms(room, janus, description) VALUES($1, $2, $3) ON CONFLICT (room) DO UPDATE SET (room, janus, description) = ($1, $2, $3) WHERE rooms.room = $1 RETURNING room",
		i.Room, i.Janus, i.Questions).Scan(&i.Room)

	if err != nil {
		return err
	}

	return nil
}

func (i *rooms) deleteRoom(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM rooms WHERE room=$1", i.Room)

	return err
}
