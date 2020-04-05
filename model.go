// model.go

package main

import (
	"database/sql"
	"encoding/json"
	"time"
)

type rooms struct {
	Room        int         `json:"room"`
	Janus       string      `json:"janus"`
	Questions   bool        `json:"questions"`
	Description string      `json:"description"`
	Num         int         `json:"num_users"`
	Users       interface{} `json:"users"`
}

type users struct {
	ID        string `json:"id"`
	Display   string `json:"display"`
	Email     string `json:"email"`
	Group     string `json:"group"`
	IP        string `json:"ip"`
	Janus     string `json:"janus"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	System    string `json:"system"`
	Username  string `json:"username"`
	Room      int    `json:"room"`
	Timestamp int    `json:"timestamp"`
	Session   int    `json:"session"`
	Handle    int    `json:"handle"`
	Rfid      int    `json:"rfid"`
	Camera    bool   `json:"camera"`
	Question  bool   `json:"question"`
	Selftest  bool   `json:"self_test"`
	Soundtest bool   `json:"sound_test"`
}

func getGroups(db *sql.DB) ([]rooms, error) {
	rows, err := db.Query("SELECT * FROM rooms ORDER BY description")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	ar := []rooms{}

	for rows.Next() {
		var r rooms

		if err := rows.Scan(&r.Room, &r.Janus, &r.Description); err != nil {
			return nil, err
		}

		rows, err := db.Query("SELECT * FROM users WHERE room = $1", r.Room)
		if err != nil {
			return nil, err
		}

		defer rows.Close()

		ur := []users{}

		for rows.Next() {
			var i users
			if err := rows.Scan(
				&i.ID,
				&i.Display,
				&i.Email,
				&i.Group,
				&i.IP,
				&i.Janus,
				&i.Name,
				&i.Role,
				&i.System,
				&i.Username,
				&i.Room,
				&i.Timestamp,
				&i.Session,
				&i.Handle,
				&i.Rfid,
				&i.Camera,
				&i.Question,
				&i.Selftest,
				&i.Soundtest); err != nil {
				return nil, err
			}
			ur = append(ur, i)
		}

		r.Users = ur
		r.Num = len(ur)

		ar = append(ar, r)
	}

	return ar, nil
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
		if err := rows.Scan(&i.Room, &i.Janus, &i.Description); err != nil {
			return nil, err
		}
		ar = append(ar, i)
	}

	return ar, nil
}

func getUsers(db *sql.DB) (map[string]interface{}, error) {
	rows, err := db.Query("SELECT * FROM users")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	json := make(map[string]interface{})

	for rows.Next() {
		var i users
		if err := rows.Scan(
			&i.ID,
			&i.Display,
			&i.Email,
			&i.Group,
			&i.IP,
			&i.Janus,
			&i.Name,
			&i.Role,
			&i.System,
			&i.Username,
			&i.Room,
			&i.Timestamp,
			&i.Session,
			&i.Handle,
			&i.Rfid,
			&i.Camera,
			&i.Question,
			&i.Selftest,
			&i.Soundtest); err != nil {
			return nil, err
		}
		json[i.ID] = i
	}

	return json, nil
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

func (i *users) getUser(db *sql.DB) error {

	err := db.QueryRow("SELECT * FROM users WHERE id = $1",
		i.ID).Scan(
		&i.ID,
		&i.Display,
		&i.Email,
		&i.Group,
		&i.IP,
		&i.Janus,
		&i.Name,
		&i.Role,
		&i.System,
		&i.Username,
		&i.Room,
		&i.Timestamp,
		&i.Session,
		&i.Handle,
		&i.Rfid,
		&i.Camera,
		&i.Question,
		&i.Selftest,
		&i.Soundtest)

	if err != nil {
		return err
	}

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

func (i *users) postUser(db *sql.DB) error {

	i.Timestamp = int(time.Now().UnixNano() / int64(time.Millisecond))
	err := db.QueryRow(
		"INSERT INTO users("+
			"id, display, email, \"group\", ip, janus, name, role, system, username, room, timestamp, session, handle, rfid, camera, question, self_test, sound_test"+
			") VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19) ON CONFLICT (id) DO UPDATE SET ("+
			"id, display, email, \"group\", ip, janus, name, role, system, username, room, timestamp, session, handle, rfid, camera, question, self_test, sound_test"+
			") = ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19) WHERE users.id = $1 RETURNING id",
		&i.ID,
		&i.Display,
		&i.Email,
		&i.Group,
		&i.IP,
		&i.Janus,
		&i.Name,
		&i.Role,
		&i.System,
		&i.Username,
		&i.Room,
		&i.Timestamp,
		&i.Session,
		&i.Handle,
		&i.Rfid,
		&i.Camera,
		&i.Question,
		&i.Selftest,
		&i.Soundtest).Scan(&i.ID)

	if err != nil {
		return err
	}

	return nil
}

func (i *rooms) deleteRoom(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM rooms WHERE room=$1", i.Room)

	return err
}

func (i *users) deleteUser(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM users WHERE id=$1", i.ID)

	return err
}
