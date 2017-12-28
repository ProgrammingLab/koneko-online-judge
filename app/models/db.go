package models

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/revel/revel"
	"golang.org/x/crypto/bcrypt"
)

var db *gorm.DB

func InitDB() {
	driver, _ := revel.Config.String("db.driver")
	spec, _ := revel.Config.String("db.spec")
	var err error
	db, err = gorm.Open(driver, spec)
	if err != nil {
		revel.AppLog.Fatal("DB Error", err.Error())
	}
	revel.AppLog.Info("DB Connected")

	db.AutoMigrate(&User{})
	db.AutoMigrate(&UserSession{})

	seed()
}

func seed() {
	password := "hoge"
	digest, _ := bcrypt.GenerateFromPassword([]byte(password), GetBcryptCost())
	user := &User{
		Name:           "test",
		DisplayName:    "test",
		Email:          "hoge@example.com",
		Authority:      authorityMember,
		PasswordDigest: string(digest),
	}
	db.Create(user)
}
