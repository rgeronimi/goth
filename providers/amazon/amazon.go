// Package amazon implements the OAuth2 protocol for authenticating users through amazon.
// This package can be used as a reference implementation of an OAuth2 provider for Goth.
package amazon

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"github.com/markbates/goth"
	"golang.org/x/oauth2"
	"log"
)

const (
	authURL            string = "https://www.amazon.com/ap/oa"
	tokenURL           string = "https://api.amazon.com/auth/o2/token"
	endpointProfile    string = "https://api.amazon.com/user/profile"
)

// Provider is the implementation of `goth.Provider` for accessing Amazon.
type Provider struct {
	ClientKey   string
	Secret      string
	CallbackURL string
	config      *oauth2.Config
}

// New creates a new Amazon provider and sets up important connection details.
// You should always call `amazon.New` to get a new provider.  Never try to
// create one manually.
func New(clientKey, secret, callbackURL string, scopes ...string) *Provider {
	p := &Provider{
		ClientKey:   clientKey,
		Secret:      secret,
		CallbackURL: callbackURL,
	}
	p.config = newConfig(p, scopes)
	return p
}

// Name is the name used to retrieve this provider later.
func (p *Provider) Name() string {
	return "amazon"
}

// Debug is a no-op for the amazon package.
func (p *Provider) Debug(debug bool) {}

// BeginAuth asks Amazon for an authentication end-point.
func (p *Provider) BeginAuth(state string) (goth.Session, error) {
	return &Session{
		AuthURL: p.config.AuthCodeURL(state),
	}, nil
}

// FetchUser will go to Amazon and access basic information about the user.
func (p *Provider) FetchUser(session goth.Session) (goth.User, error) {
	sess := session.(*Session)
	user := goth.User{
		AccessToken: sess.AccessToken,
		Provider:    p.Name(),
	}

	response, err := http.Get(endpointProfile + "?access_token=" + url.QueryEscape(sess.AccessToken))
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		return user, err
	}
	defer response.Body.Close()

	bits, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return user, err
	}

	err = json.NewDecoder(bytes.NewReader(bits)).Decode(&user.RawData)
	if err != nil {
		return user, err
	}

	err = userFromReader(bytes.NewReader(bits), &user)
	log.Println(user)
	return user, err
}
// UnmarshalSession wil unmarshal a JSON string into a session.
func (p *Provider) UnmarshalSession(data string) (goth.Session, error) {
	s := &Session{}
	err := json.NewDecoder(strings.NewReader(data)).Decode(s)
	return s, err
}

func newConfig(provider *Provider, scopes []string) *oauth2.Config {
	c := &oauth2.Config{
		ClientID:     provider.ClientKey,
		ClientSecret: provider.Secret,
		RedirectURL:  provider.CallbackURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
		Scopes: []string{},
	}

	if len(scopes) > 0 {
		for _, scope := range scopes {
			c.Scopes = append(c.Scopes, scope)
		}
	}else {
		c.Scopes = append(c.Scopes, "profile", "postal_code")
	}
	return c
}

func userFromReader(r io.Reader, user *goth.User) error {
	u := struct {
		Name     string `json:"name"`
		Location string `json:"postal_code"`
		Email    string `json:"email"`
		Id       string `json:"user_id"`
	}{}
	err := json.NewDecoder(r).Decode(&u)
	if err != nil {
		return err
	}
	user.Email = u.Email
	user.Name = u.Name
	user.NickName = u.Name
	user.UserID = u.Id
	user.Location=u.Location
	return nil
}