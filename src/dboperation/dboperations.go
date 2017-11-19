package dboperation

import "database/sql"
import (
	_ "github.com/go-sql-driver/mysql"
	"Models"
)


var createTable string  = "CREATE TABLE User (ID INT NOT NULL, Username TEXT(100) NOT NULL, FullName TEXT(100) NOT NULL, " +
	"email TEXT(150) NOT NULL, PasswordHash TEXT(500) NOT NULL, jwtToken TEXT(500) NOT NULL, IsDisabled BOOL NOT NULL, " +
	"PRIMARY KEY (ID), UNIQUE INDEX ID_UNIQUE (ID ASC));"


func UserRegistration(userModel Models.User) {

}

func CreateDatabase() bool {
	db, err := sql.Open("mysql", "root:ZXCfdsa1208@tcp(18.195.2.253:3306)/TaskCalendarDb")
	if err != nil {
		panic(err.Error())
	}

	ret, err := db.Exec(createTable)
	if err != nil{
		panic(ret)
	}

	return true
}

func WriteDataToDb(user Models.User) bool {
	db, err := sql.Open("mysql", "root:ZXCfdsa1208@tcp(18.195.2.253:3306)/TaskCalendarDb")
	if err != nil {
		panic(err.Error())
		return false
	}
	defer db.Close()

	stmIns, err := db.Prepare("INSERT INTO User (Username, FullName, email, PasswordHash, jwtToken, IsDisabled) VALUES (?, ?, ?, ?, ?)")
	defer stmIns.Close()

	_, err = stmIns.Exec(user.Username, user.FullName, user.Email, user.PasswordHash, false)
	if err != nil {
		panic(err.Error())
		return false
	}

	return true
}

func GetHashFromDb(user Models.User) (string, bool) {
	db, err := sql.Open("mysql", "root:ZXCfdsa1208@tcp(18.195.2.253:3306)/TaskCalendarDb")

	if err != nil {
		panic(err.Error())
		return "", false
	}
	defer db.Close()

	sel, err := db.Prepare("SELECT PasswordHash FROM User WHERE Username = ?;")
	if err != nil {
		panic(err.Error())
		return "", false
	}
	defer sel.Close()

	var hash string
	err = sel.QueryRow(user.Username).Scan(&hash)
	if err != nil {
		panic(err.Error())
		return "", false
	}

	return hash, true
}

func UpdateTokenForUser(user Models.User) bool {
	db, err := sql.Open("mysql", "root:ZXCfdsa1208@tcp(18.195.2.253:3306)/TaskCalendarDb")

	if err != nil {
		panic(err.Error())
		return false
	}
	defer db.Close()

	upd, err := db.Prepare("UPDATE User SET jwtToken = ? WHERE Username = ?;")
	if err != nil {
		panic(err.Error())
		return false
	}

	defer upd.Close()

	_, err = upd.Exec(user.JsonToken, user.Username)
	if err != nil {
		panic(err.Error())
		return false
	}

	return true
}

func CheckifUserExist(user Models.User) bool {
	db, err := sql.Open("mysql", "root:ZXCfdsa1208@tcp(18.195.2.253:3306)/TaskCalendarDb")
	if err != nil {
		panic(err.Error())
		return false
	}
	defer db.Close()

	sel, err := db.Prepare("SELECT ID FROM User WHERE Username = ?;")
	if err != nil {
		panic(err.Error())
		return false
	}
	defer sel.Close()

	var id int

	err = sel.QueryRow(user.Username).Scan(&id)
	if err != nil { //NoRows error - is good, user does no exist
		return false
	} else {
		return true // else - user exist
	}

}







































