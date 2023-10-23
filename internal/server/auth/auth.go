package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Jourloy/jourloy-hh/internal/server/storage"
	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
)

var (
	clientID     string
	clientSecret string
	redirectURI  string
	logger       = log.NewWithOptions(os.Stderr, log.Options{Prefix: `[auth]`, Level: log.DebugLevel})
)

type AuthService struct {
	database storage.PostgresRepository
}

func NewAuthService(d storage.PostgresRepository) *AuthService {
	id, exist := os.LookupEnv(`HH_CLIENT_ID`)
	if !exist {
		logger.Fatal(`Error loading HH_CLIENT_ID environment variable`)
	}
	clientID = id

	secret, exist := os.LookupEnv(`HH_CLIENT_SECRET`)
	if !exist {
		logger.Fatal(`Error loading HH_CLIENT_SECRET environment variable`)
	}
	clientSecret = secret

	redirect, exist := os.LookupEnv(`HH_REDIRECT_URI`)
	if !exist {
		logger.Fatal(`Error loading HH_REDIRECT_URI environment variable`)
	}
	redirectURI = redirect

	return &AuthService{
		database: d,
	}
}

func (a *AuthService) Redirect(ctx *gin.Context) {
	uri := `https://hh.ru/oauth/authorize`
	params := url.Values{
		`response_type`: {`code`},
		`client_id`:     {clientID},
		`redirect_uri`:  {redirectURI},
	}

	logger.Debug(`Redirecting to ` + uri + `?` + params.Encode())

	ctx.Redirect(302, uri+`?`+params.Encode())
}

type HHTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
}

type HHResume struct {
	Age       int    `json:"age"`
	Id        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type HHResumeResponse struct {
	Found int        `json:"found"`
	Items []HHResume `json:"items"`
}

func (a *AuthService) Callback(ctx *gin.Context) {
	// Check code
	code := ctx.Query(`code`)
	if code == `` {
		logger.Error(`Callback code is empty`)
		return
	}

	// Get access and refresh token
	uri := `https://hh.ru/oauth/token`
	body := url.Values{
		`grant_type`:    {`authorization_code`},
		`code`:          {code},
		`client_id`:     {clientID},
		`client_secret`: {clientSecret},
		`redirect_uri`:  {redirectURI},
	}
	resp, err := http.Post(uri, `application/x-www-form-urlencoded`, strings.NewReader(body.Encode()))
	if err != nil {
		logger.Error(err)
		ctx.String(400, err.Error())
	}

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err)
		ctx.String(400, err.Error())
		return
	}
	defer resp.Body.Close()

	// Unmarshal
	response := HHTokenResponse{}
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		logger.Error(err)
		ctx.String(400, err.Error())
		return
	}

	// Prepare request for getting resume
	req, err := http.NewRequest(`GET`, `https://api.hh.ru/resumes/mine`, nil)
	if err != nil {
		logger.Error(err)
		ctx.String(400, err.Error())
		return
	}
	req.Header.Add(`Authorization`, `Bearer `+response.AccessToken)
	req.Header.Add(`HH-User-Agent`, `HHelper/1.0 (jourloy@yandex.ru)`)
	req.Header.Add(`Accept`, `application/json`)

	// Get resume
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error(err)
		ctx.String(400, err.Error())
		return
	}

	// Read response
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error(err)
		ctx.String(400, err.Error())
		return
	}
	defer res.Body.Close()

	// Unmarshal
	var jsonBody HHResumeResponse
	err = json.Unmarshal([]byte(resBody), &jsonBody)
	if err != nil {
		logger.Error(err)
		ctx.String(400, err.Error())
		return
	}

	// Save user
	err = a.database.CreateUser(jsonBody.Items[0].Id, response.AccessToken, response.RefreshToken, code, response.ExpiresIn)
	if err != nil {
		logger.Error(err)
		return
	}

	ctx.JSON(200, gin.H{
		`status`: `ok`,
	})
}
