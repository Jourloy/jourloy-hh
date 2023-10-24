package parser

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Jourloy/jourloy-hh/internal/server/storage"
	"github.com/charmbracelet/log"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{Prefix: `[parser]`, Level: log.DebugLevel})
)

type ParserService struct {
	database storage.PostgresRepository
	done     chan struct{}
}

func NewParserService(d storage.PostgresRepository) *ParserService {
	return &ParserService{
		database: d,
		done:     make(chan struct{}),
	}
}

func (p *ParserService) StartTickers() {
	parseTicker := time.NewTicker(10 * time.Minute)
	tokenTicker := time.NewTicker(1 * time.Minute)

	logger.Debug(`Tickers started`)

	for {
		select {
		case <-p.done:
			return

		case <-parseTicker.C:
			p.Parse()

		case <-tokenTicker.C:
			p.UpdateTokens()
		}
	}
}

func (p *ParserService) StopTickers() {
	p.done <- struct{}{}
}

type Salary struct {
	Currency *string `json:"currency"`
	From     *int    `json:"from"`
	To       *int    `json:"to"`
	Gross    *bool   `json:"gross"`
}

type Phone struct {
	Number  string `json:"number"`
	Country string `json:"country"`
	City    string `json:"city"`
	Comment string `json:"comment"`
}

type Contacts struct {
	Email  *string `json:"email"`
	Name   *string `json:"name"`
	Phones []Phone `json:"phones"`
}

type Counters struct {
	Responses int `json:"responses"`
}

type Vacancy struct {
	Name         string    `json:"name"`
	PublishedAt  string    `json:"published_at"`
	AlternateURL string    `json:"alternate_url"`
	VacancyID    string    `json:"id"`
	URL          string    `json:"url"`
	Salary       *Salary   `json:"salary"`
	Contacts     *Contacts `json:"contacts"`
	Counters     *Counters `json:"counters"`
}

type VacancyResponse struct {
	Found int       `json:"found"`
	Items []Vacancy `json:"items"`
}

func (p *ParserService) Parse() {
	logger.Debug(`Parsing vacancies...`)

	// Get all users
	users, err := p.database.FindAllUsers()
	if err != nil {
		logger.Error(err)
		return
	}

	// Parse resumes with Nest.JS
	for _, user := range users {
		if (user.AddedAt + user.ExpiresIn) < int(time.Now().Unix()) {
			continue
		}

		res := p.GetNestResumes(user)
		if res == nil {
			continue
		}

		p.AddVacancy(res)
	}

	// Parse resumes with Go
	for _, user := range users {
		if (user.AddedAt + user.ExpiresIn) < int(time.Now().Unix()) {
			continue
		}

		res := p.GetGoResumes(user)
		if res == nil {
			continue
		}

		p.AddVacancy(res)
	}
}

func (p *ParserService) AddVacancy(res *http.Response) {
	// Read response
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error(err)
		return
	}
	defer res.Body.Close()

	// Unmarshal
	var jsonBody VacancyResponse
	err = json.Unmarshal([]byte(resBody), &jsonBody)
	if err != nil {
		logger.Error(err)
		return
	}

	// Check body
	if jsonBody.Found == 0 {
		return
	}

	// Add vacancy
	for _, v := range jsonBody.Items {
		// If vacancy already exists then skip
		if _, err := p.database.FindVacancyByID(v.VacancyID); err == nil {
			continue
		}

		var SalaryCurrency string
		if v.Salary != nil && v.Salary.Currency != nil {
			SalaryCurrency = *v.Salary.Currency
		} else {
			SalaryCurrency = `Null`
		}

		var SalaryFrom int
		if v.Salary != nil && v.Salary.From != nil {
			SalaryFrom = *v.Salary.From
		} else {
			SalaryFrom = 0
		}

		var SalaryTo int
		if v.Salary != nil && v.Salary.To != nil {
			SalaryTo = *v.Salary.To
		} else {
			SalaryTo = 0
		}

		var ContactsEmail string
		if v.Contacts != nil && v.Contacts.Email != nil {
			ContactsEmail = *v.Contacts.Email
		} else {
			ContactsEmail = `Null`
		}

		var ContactsName string
		if v.Contacts != nil && v.Contacts.Name != nil {
			ContactsName = *v.Contacts.Name
		} else {
			ContactsName = `Null`
		}

		// Create vacancy
		vacancy := storage.VacancyModel{
			Name:           v.Name,
			AlternateURL:   v.AlternateURL,
			VacancyID:      v.VacancyID,
			URL:            v.URL,
			SalaryCurrency: SalaryCurrency,
			SalaryFrom:     SalaryFrom,
			SalaryTo:       SalaryTo,
			ContactsEmail:  ContactsEmail,
			ContactsName:   ContactsName,
			Notificated:    false,
		}

		err = p.database.CreateVacancy(vacancy)
		if err != nil {
			logger.Error(err)
			continue
		}
	}
}

func (p *ParserService) GetGoResumes(user storage.UserModel) *http.Response {
	uri := `https://api.hh.ru/resumes/` + user.ResumeID + `/similar_vacancies`

	nestValues := url.Values{
		`per_page`: {`100`},
		`text`:     {`Go`},
	}
	req, err := http.NewRequest(`GET`, uri+`?`+nestValues.Encode(), nil)
	req.Header.Add(`Authorization`, `Bearer `+user.AccessToken)
	req.Header.Add(`HH-User-Agent`, `HHelper/1.0 (jourloy@yandex.ru)`)
	req.Header.Add(`Accept`, `application/json`)
	if err != nil {
		logger.Error(err)
		return nil
	}

	// Get resume
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error(err)
		return nil
	}

	return res
}

func (p *ParserService) GetNestResumes(user storage.UserModel) *http.Response {
	uri := `https://api.hh.ru/resumes/` + user.ResumeID + `/similar_vacancies`

	nestValues := url.Values{
		`per_page`: {`100`},
		`text`:     {`Nest.js`},
	}
	req, err := http.NewRequest(`GET`, uri+`?`+nestValues.Encode(), nil)
	req.Header.Add(`Authorization`, `Bearer `+user.AccessToken)
	req.Header.Add(`HH-User-Agent`, `HHelper/1.0 (jourloy@yandex.ru)`)
	req.Header.Add(`Accept`, `application/json`)
	if err != nil {
		logger.Error(err)
		return nil
	}

	// Get resume
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error(err)
		return nil
	}

	return res
}

type HHTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
}

func (p *ParserService) UpdateTokens() {
	logger.Debug(`Update tokens...`)

	users, err := p.database.FindAllUsers()
	if err != nil {
		logger.Error(err)
		return
	}

	for _, user := range users {
		if (user.AddedAt + user.ExpiresIn) > int(time.Now().Unix()) {
			logger.Debug(`Skip token update for user: ` + user.ResumeID)
			continue
		}

		uri := `https://api.hh.ru/oauth/token`
		body := url.Values{
			`grant_type`:    {`refresh_token`},
			`refresh_token`: {user.RefreshToken},
		}

		resp, err := http.Post(uri, `application/x-www-form-urlencoded`, strings.NewReader(body.Encode()))
		if err != nil {
			logger.Error(err)
		}

		// Read response
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error(err)
			return
		}
		defer resp.Body.Close()

		// Unmarshal
		response := HHTokenResponse{}
		err = json.Unmarshal(respBody, &response)
		if err != nil {
			logger.Error(err)
			return
		}

		user.AccessToken = response.AccessToken
		user.RefreshToken = response.RefreshToken
		user.ExpiresIn = response.ExpiresIn
		user.AddedAt = int(time.Now().Unix())

		if err := p.database.UpdateUser(user); err != nil {
			logger.Error(err)
			return
		}
	}
}
