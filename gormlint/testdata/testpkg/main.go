package testpkg

import "gorm.io/gorm"

type User struct{ ID int }

func good(db *gorm.DB) {
	var u User
	db.First(&u)
}

func bad(db *gorm.DB) {
	var u User
	db.First(&u, 10) // want "gorm DB.First must be called with exactly 1 argument"
}

// no runtime setup needed for analysistest
