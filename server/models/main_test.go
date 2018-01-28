package models

import (
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestMain(m *testing.M) {
	var (
		driver = DefaultString(os.Getenv("KOJ_DB_DRIVER"), "mysql")
		spec   = DefaultString(os.Getenv("KOJ_DB_TEST_SPEC"), "root:@/koj_test?charset=utf8&parseTime=True&loc=Local")
	)

	if err := connectDB(driver, spec); err != nil {
		panic(err)
	}

	db.LogMode(true)

	createTables()
	seedLanguages()
	insertTestUsers()

	os.Exit(m.Run())
}

func insertTestUsers() {
	password, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	admin := &User{
		Name:           "admin",
		DisplayName:    "admin",
		Email:          "admin@example.com",
		Authority:      Admin,
		PasswordDigest: string(password),
	}
	insertUserIfNonExisting(admin)
	test := &User{
		Name:           "test",
		DisplayName:    "test",
		Email:          "test@example.com",
		Authority:      Member,
		PasswordDigest: string(password),
	}
	insertUserIfNonExisting(test)
}
