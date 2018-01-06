package models

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/revel/revel"
)

var db *gorm.DB

func InitDB() {
	driver, _ := revel.Config.String("db.driver")
	spec, _ := revel.Config.String("db.spec")
	var err error
	db, err = gorm.Open(driver, spec)
	if err != nil {
		revel.AppLog.Fatal("DB Error", err.Error())
		panic(err)
	}
	revel.AppLog.Info("DB Connected")
	if revel.DevMode {
		db.LogMode(true)
	}

	createTables()
	seedLanguages()
	insertAdmin()
}

func createTables() {
	// リファレンス通りにやってもforeign keyにならなかったので、自分でAddForeignKeyしてる
	db.AutoMigrate(&User{})
	db.AutoMigrate(&UserSession{})
	db.Model(&UserSession{}).AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT")

	db.AutoMigrate(&Problem{})
	db.Model(&Problem{}).AddForeignKey("writer_id", "users(id)", "RESTRICT", "RESTRICT")
	db.AutoMigrate(&Sample{})
	db.Model(&Sample{}).AddForeignKey("problem_id", "problems(id)", "RESTRICT", "RESTRICT")

	db.AutoMigrate(&CaseSet{})
	db.Model(&CaseSet{}).AddForeignKey("problem_id", "problems(id)", "RESTRICT", "RESTRICT")
	db.AutoMigrate(&TestCase{})
	db.Model(&TestCase{}).AddForeignKey("case_set_id", "case_sets(id)", "RESTRICT", "RESTRICT")

	db.AutoMigrate(&Language{})
	db.AutoMigrate(&Submission{})
	db.Model(&Submission{}).AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT")
	db.Model(&Submission{}).AddForeignKey("language_id", "languages(id)", "RESTRICT", "RESTRICT")
	db.Model(&Submission{}).AddForeignKey("problem_id", "problems(id)", "RESTRICT", "RESTRICT")

	db.AutoMigrate(&JudgeSetResult{})
	db.AutoMigrate(&JudgeResult{})
	db.Model(&JudgeSetResult{}).AddForeignKey("submission_id", "submissions(id)", "RESTRICT", "RESTRICT")
	db.Model(&JudgeSetResult{}).AddForeignKey("case_set_id", "case_sets(id)", "RESTRICT", "RESTRICT")
	db.Model(&JudgeResult{}).AddForeignKey("judge_set_result_id", "judge_set_results(id)", "RESTRICT", "RESTRICT")
	db.Model(&JudgeResult{}).AddForeignKey("test_case_id", "test_cases(id)", "RESTRICT", "RESTRICT")

	db.AutoMigrate(&Contest{})
	db.Model(&Problem{}).AddForeignKey("contest_id", "contests(id)", "RESTRICT", "RESTRICT")
	db.Table("contests_writers").AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT")
	db.Table("contests_writers").AddForeignKey("contest_id", "contests(id)", "RESTRICT", "RESTRICT")
	db.Table("contests_participants").AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT")
	db.Table("contests_participants").AddForeignKey("contest_id", "contests(id)", "RESTRICT", "RESTRICT")
}

func seedLanguages() {
	languages := []*Language{
		{
			ImageName:      "cpp",
			DisplayName:    "C++17 (GCC 7.2.0)",
			FileName:       "main.cpp",
			ExeFileName:    "main.o",
			CompileCommand: "g++ -w -lm -std=gnu++17 -O2 -o main.o main.cpp",
			ExecCommand:    "./main.o",
		},
		{
			ImageName:      "cpp",
			DisplayName:    "C++11 (GCC 7.2.0)",
			FileName:       "main.cpp",
			ExeFileName:    "main.o",
			CompileCommand: "g++ -w -lm -std=gnu++11 -O2 -o main.o main.cpp",
			ExecCommand:    "./main.o",
		},
		{
			ImageName:      "cpp",
			DisplayName:    "C11 (GCC 7.2.0)",
			FileName:       "main.c",
			ExeFileName:    "main.o",
			CompileCommand: "gcc -w -lm -std=gnu11 -O2 -o main.o main.c",
			ExecCommand:    "./main.o",
		},
	}

	for _, l := range languages {
		db.Save(l)
	}
}

func insertAdmin() {
	admin := &User{
		Name:           "admin",
		DisplayName:    "admin",
		Email:          "admin@judge.kurume-nct.com",
		Authority:      authorityAdmin,
		PasswordDigest: GenerateRandomBase64String(64),
	}
	insertUserIfNonExisting(admin)
}

func insertUserIfNonExisting(user *User) {
	existing := &User{}
	if db.Where("name = ?", user.Name).First(existing).RecordNotFound() {
		db.Save(user)
	}
}
