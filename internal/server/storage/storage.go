package storage

import (
	"os"

	"github.com/charmbracelet/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	database *gorm.DB
	logger   = log.NewWithOptions(os.Stderr, log.Options{Prefix: `[postgres]`, Level: log.DebugLevel})
)

type User struct {
	gorm.Model
	ResumeID     string
	Code         string
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type PostgresRepository struct{}

func NewRepository() *PostgresRepository {
	databaseURL, exist := os.LookupEnv(`DATABASE_URL`)
	if !exist {
		logger.Fatal(`Error loading DATABASE_URL environment variable`)
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		logger.Fatal(err)
	}
	database = db

	database.AutoMigrate(&User{})

	return &PostgresRepository{}
}

func (p *PostgresRepository) CreateUser(id string, accessToken string, refreshToken string, code string, expiresIn int) error {
	user := User{
		ResumeID:     id,
		Code:         code,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}
	return database.Create(&user).Error
}

func (p *PostgresRepository) FindAllUsers() ([]User, error) {
	var users []User
	return users, database.Find(&users).Error
}

func (p *PostgresRepository) UpdateUser(user User) error {
	return database.Save(&user).Error
}
