// Package box implements the OAuth2 protocol for authenticating users through box.
// This package can be used as a reference implementation of an OAuth2 provider for Goth.
package box

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"github.com/markbates/goth"
	"golang.org/x/oauth2"
)

const (
	authURL            string = "https://app.box.com/api/oauth2/authorize"
	tokenURL           string = "https://app.box.com/api/oauth2/token"
	endpointProfile    string = "https://api.box.com/2.0/users/me"
)

// Provider is the implementation of `goth.Provider` for accessing Box.
type Provider struct {
	ClientKey   string
	Secret      string
	CallbackURL string
	config      *oauth2.Config
}

// New creates a new Box provider and sets up important connection details.
// You should always call `box.New` to get a new provider.  Never try to
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
	return "box"
}

// Debug is a no-op for the box package.
func (p *Provider) Debug(debug bool) {}

// BeginAuth asks Box for an authentication end-point.
func (p *Provider) BeginAuth(state string) (goth.Session, error) {
	return &Session{
		AuthURL: p.config.AuthCodeURL(state),
	}, nil
}

// FetchUser will go to Box and access basic information about the user.
func (p *Provider) FetchUser(session goth.Session) (goth.User, error) {
	s := session.(*Session)
	user := goth.User{
		AccessToken: s.AccessToken,
		Provider:    p.Name(),
	}
	req, err := http.NewRequest("GET", endpointProfile, nil)
	if err != nil {
		return user, err
	}
	req.Header.Set("Authorization", "Bearer "+s.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return user, err
	}
	defer resp.Body.Close()

	err = userFromReader(resp.Body, &user)
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
	}

	return c
}

func userFromReader(r io.Reader, user *goth.User) error {
	u := struct {
		Name        string `json:"name"`
		Location    string `json:"address"`
		Email       string `json:"login"`
		AvatarURL   string `json:"avatar_url"`
		Id          string `json:"id"`
	}{}
	err := json.NewDecoder(r).Decode(&u)
	if err != nil {
		return err
	}
	user.Email = u.Email
	user.Name = u.Name
	user.NickName = u.Name
	user.UserID = u.Id
	user.Location = u.Location
	return nil
}