package web

import (
	"testing"
	"github.com/cleverua/tuna-timer-api/utils"
	"github.com/cleverua/tuna-timer-api/data"
	"github.com/cleverua/tuna-timer-api/models"
	"github.com/nlopes/slack"
	"gopkg.in/mgo.v2/bson"
	"log"
	"gopkg.in/tylerb/is.v1"
	"gopkg.in/mgo.v2"
	"github.com/pavlo/gosuite"
	"net/url"
	"net/http"
	"bytes"
	"net/http/httptest"
	"encoding/json"
	"time"
)

func TestFrontendHandlers(t *testing.T) {
	gosuite.Run(t, &FrontendHandlersTestSuite{Is: is.New(t)})
}

func (s *FrontendHandlersTestSuite) TestUserAuthentication(t *testing.T) {
	s.urlValues.Set("pid", "pass-for-jwt-generation")
	req, err := http.NewRequest("POST", "/frontend/sessions", bytes.NewBufferString(s.urlValues.Encode()))
	s.Nil(err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;")

	h := NewFrontendHandlers(s.env, s.session)
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(h.UserAuthentication)
	handler.ServeHTTP(recorder, req)

	resp := JwtResponseBody{ResponseData: JwtToken{}}
	err = json.Unmarshal(recorder.Body.Bytes(), &resp)
	s.Nil(err)

	verificationToken, err := NewUserToken(s.user.ID.Hex(), s.session)
	s.Nil(err)
	s.Equal(resp.ResponseErrors, make(map[string]string))
	s.Equal(resp.ResponseData.Token, verificationToken)
}

func (s *FrontendHandlersTestSuite) TestUserAuthenticationWithWrongPid(t *testing.T) {
	s.urlValues.Set("pid", "gIkuvaNzQIHg97ATvDxqgjtO")

	req, err := http.NewRequest("POST", "/api/v1/frontend/session", bytes.NewBufferString(s.urlValues.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;")
	s.Nil(err)

	h := NewFrontendHandlers(s.env, s.session)
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(h.UserAuthentication)
	handler.ServeHTTP(recorder, req)

	resp := JwtResponseBody{}
	err = json.Unmarshal(recorder.Body.Bytes(), &resp)
	s.Nil(err)
	s.Equal(resp.ResponseErrors["userMessage"], "please login from slack application")
	s.Equal(resp.ResponseData.Token, "")
	s.Equal(resp.AppInfo["env"], utils.TestEnv)
	s.Equal(resp.AppInfo["version"], s.env.AppVersion)
}

type FrontendHandlersTestSuite struct {
	*is.Is
	env        *utils.Environment
	session    *mgo.Session
	user       *models.TeamUser
	pass       *models.Pass
	urlValues  url.Values
}

func (s *FrontendHandlersTestSuite) SetUpSuite() {
	e := utils.NewEnvironment(utils.TestEnv, "1.0.0")

	session, err := utils.ConnectToDatabase(e.Config)
	if err != nil {
		log.Fatal("Failed to connect to DB!")
	}

	s.session = session.Clone()
	e.MigrateDatabase(session)
	s.env = e
}

func (s *FrontendHandlersTestSuite) TearDownSuite() {
	s.session.Close()
}

func (s *FrontendHandlersTestSuite) SetUp() {
	s.urlValues = url.Values{}

	//Clear Database
	utils.TruncateTables(s.session)

	//Seed Database
	passRepository := data.NewPassRepository(s.session)
	userRepository := data.NewUserRepository(s.session)
	var err error
	s.user = &models.TeamUser{
		TeamID:           "team-id",
		ExternalUserID:   "ext-user-id",
		ExternalUserName: "user-name",
		SlackUserInfo:    &slack.User{
			IsAdmin: true,
		},
	}
	_, err = userRepository.Save(s.user)
	s.Nil(err)

	s.pass = &models.Pass{
		ID:           bson.NewObjectId(),
		Token:        "pass-for-jwt-generation",
		TeamUserID:   s.user.ID.Hex(),
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(5 * time.Minute),
	}
	err = passRepository.Insert(s.pass)
	s.Nil(err)
}

func (s *FrontendHandlersTestSuite) TearDown() {}
