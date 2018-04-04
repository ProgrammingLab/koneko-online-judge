package models

import (
	"fmt"

	"github.com/gedorinku/koneko-online-judge/server/conf"
	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"golang.org/x/crypto/bcrypt"
)

var db *gorm.DB

func InitDB() {
	const driver = "mysql"
	cfg := conf.GetConfig().Koneko
	user := cfg.DBUser
	pass := cfg.DBPassword
	host := cfg.DBHost
	name := cfg.DBName
	spec := fmt.Sprintf("%v:%v@tcp(%v)/%v?charset=utf8&parseTime=True&loc=Local", user, pass, host, name)

	err := connectDB(driver, spec)
	if err != nil {
		logger.AppLog.Fatal("DB Error", err.Error())
		panic(err)
	}

	logger.AppLog.Info("DB Connected")
	db.LogMode(true)

	createTables()
	seedLanguages()
	insertAdmin()
}

func connectDB(driver, spec string) error {
	// model.dbに代入したいので。
	var err error
	db, err = gorm.Open(driver, spec)
	return err
}

func createTables() {
	// リファレンス通りにやってもforeign keyにならなかったので、自分でAddForeignKeyしてる
	db.AutoMigrate(&User{})
	db.AutoMigrate(&UserSession{})
	db.Model(&UserSession{}).AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT")

	db.AutoMigrate(&JudgementConfig{})

	db.AutoMigrate(&Problem{})
	db.Model(&Problem{}).AddForeignKey("writer_id", "users(id)", "RESTRICT", "RESTRICT")
	db.Model(&Problem{}).AddForeignKey("judgement_config_id", "judgement_configs(id)", "RESTRICT", "RESTRICT")
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
	db.Model(&JudgementConfig{}).AddForeignKey("language_id", "languages(id)", "RESTRICT", "RESTRICT")

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

	db.AutoMigrate(&Score{})
	db.Model(&Score{}).AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT")
	db.Model(&Score{}).AddForeignKey("contest_id", "contests(id)", "RESTRICT", "RESTRICT")

	db.AutoMigrate(&ScoreDetail{})
	db.Model(&ScoreDetail{}).AddForeignKey("score_id", "scores(id)", "RESTRICT", "RESTRICT")
	db.Model(&ScoreDetail{}).AddForeignKey("problem_id", "problems(id)", "RESTRICT", "RESTRICT")

	db.AutoMigrate(&PasswordResetToken{})
	db.Model(&PasswordResetToken{}).AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT")

	db.AutoMigrate(&WhiteEmail{})
	db.Model(&WhiteEmail{}).AddForeignKey("created_by_id", "users(id)", "RESTRICT", "RESTRICT")

	db.AutoMigrate(&EmailConfirmation{})
	db.Model(&EmailConfirmation{}).AddForeignKey("white_email_id", "white_emails(id)", "RESTRICT", "RESTRICT")
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
	const email = "admin@judge.kurume-nct.com"
	password := GenerateRandomBase64String(12)

	digest, err := bcrypt.GenerateFromPassword([]byte(password), GetBcryptCost())
	if err != nil {
		panic(err)
	}

	admin := &User{
		Name:           "admin",
		DisplayName:    "admin",
		Email:          email,
		Authority:      Admin,
		PasswordDigest: string(digest),
	}
	inserted := insertUserIfNonExisting(admin)
	if inserted {
		logger.AppLog.Infof("admin user -> email: %v password: %v", email, password)
	}
}

func insertUserIfNonExisting(user *User) bool {
	existing := &User{}
	if db.Where("name = ?", user.Name).First(existing).RecordNotFound() {
		db.Save(user)
		return true
	}
	return false
}
