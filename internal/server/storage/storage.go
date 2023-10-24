package storage

import (
	"os"
	"time"

	"github.com/charmbracelet/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	database *gorm.DB
	logger   = log.NewWithOptions(os.Stderr, log.Options{Prefix: `[postgres]`, Level: log.DebugLevel})
)

type UserModel struct {
	gorm.Model
	ResumeID     string
	Code         string
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	AddedAt      int
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

	database.AutoMigrate(&UserModel{}, &VacancyModel{})

	return &PostgresRepository{}
}

func (p *PostgresRepository) CreateUser(id string, accessToken string, refreshToken string, code string, expiresIn int) error {
	user := UserModel{
		ResumeID:     id,
		Code:         code,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		AddedAt:      int(time.Now().Unix()),
	}
	return database.Create(&user).Error
}

func (p *PostgresRepository) FindAllUsers() ([]UserModel, error) {
	var users []UserModel
	return users, database.Find(&users).Error
}

func (p *PostgresRepository) UpdateUser(user UserModel) error {
	return database.Save(&user).Error
}

type VacancyModel struct {
	gorm.Model
	VacancyID      string
	Name           string
	URL            string
	AlternateURL   string
	SalaryCurrency string
	SalaryFrom     int
	SalaryTo       int
	ContactsEmail  string
	ContactsName   string
	Notificated    bool
}

func (p *PostgresRepository) CreateVacancy(vacancy VacancyModel) error {
	return database.Create(&vacancy).Error
}

func (p *PostgresRepository) FindAllVacancies() ([]VacancyModel, error) {
	var vacancies []VacancyModel
	return vacancies, database.Find(&vacancies).Error
}

func (p *PostgresRepository) FindVacancyByID(id string) (VacancyModel, error) {
	var vacancy VacancyModel
	return vacancy, database.First(&vacancy, "vacancy_id = ?", id).Error
}

func (p *PostgresRepository) UpdateVacancy(vacancy VacancyModel) error {
	return database.Save(&vacancy).Error
}
