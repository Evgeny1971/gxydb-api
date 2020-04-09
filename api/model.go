package api

import (
	"time"

	"github.com/volatiletech/sqlboiler/boil"
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

func getGroups(exec boil.Executor) ([]rooms, error) {
	rows, err := exec.Query("SELECT * FROM rooms ORDER BY description")

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

		rows, err := exec.Query("SELECT * FROM users WHERE room = $1", r.Room)
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

func getRooms(exec boil.Executor) ([]rooms, error) {
	rows, err := exec.Query("SELECT * FROM rooms ORDER BY description")

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

func getUsers(exec boil.Executor) (map[string]interface{}, error) {
	rows, err := exec.Query("SELECT * FROM users")

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

func (i *rooms) getRoom(exec boil.Executor) error {
	return exec.QueryRow("SELECT janus, description FROM rooms WHERE room = $1", i.Room).
		Scan(&i.Janus, &i.Description)
}

func (i *users) getUser(exec boil.Executor) error {

	err := exec.QueryRow("SELECT * FROM users WHERE id = $1",
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

func (i *rooms) postRoom(exec boil.Executor) error {

	err := exec.QueryRow(
		"INSERT INTO rooms(room, janus, description) VALUES($1, $2, $3) ON CONFLICT (room) DO UPDATE SET (room, janus, description) = ($1, $2, $3) WHERE rooms.room = $1 RETURNING room",
		i.Room, i.Janus, i.Description).Scan(&i.Room)

	if err != nil {
		return err
	}

	return nil
}

func (i *users) postUser(exec boil.Executor) error {

	i.Timestamp = int(time.Now().UnixNano() / int64(time.Millisecond))
	err := exec.QueryRow(
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

func (i *rooms) deleteRoom(exec boil.Executor) error {
	_, err := exec.Exec("DELETE FROM rooms WHERE room=$1", i.Room)

	return err
}

func (i *users) deleteUser(exec boil.Executor) error {
	_, err := exec.Exec("DELETE FROM users WHERE id=$1", i.ID)

	return err
}
