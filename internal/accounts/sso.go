package accounts

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/internal/sso"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// ErrNoAccount is returned for a person who signed in with a provider but
// has no account and may not get one.
var ErrNoAccount = errors.New("no account")

// SignInWith returns the account of someone who signed in with a provider:
//
//  1. the account linked to their identity;
//  2. else the account with their verified email, which gets linked;
//  3. else, for an allowed domain, a new account with the provider's
//     default role.
func SignInWith(p *sso.Provider, id sso.Identity) (user *models.User, created bool, err error) {
	if u, err := storage.GetUserBySSO(id.Provider, id.Subject); err == nil {
		return u, false, nil
	} else if !errors.Is(err, storage.ErrUserNotFound) {
		return nil, false, err
	}
	if !id.EmailVerified || id.Email == "" {
		return nil, false, fmt.Errorf("%w: %s did not confirm an email address for this account", ErrNoAccount, p.Label)
	}
	if u, err := storage.GetUserByEmail(id.Email); err == nil {
		if u.SSOProvider == id.Provider && u.SSOSubject != "" && u.SSOSubject != id.Subject {
			return nil, false, fmt.Errorf("%w: %s is linked to another %s account", ErrNoAccount, id.Email, p.Label)
		}
		if err := storage.LinkUserSSO(u.ID, id.Provider, id.Subject); err != nil {
			return nil, false, err
		}
		u.SSOProvider, u.SSOSubject = id.Provider, id.Subject
		logger.Log.WithFields(map[string]any{"username": u.Username, "provider": id.Provider}).Info("Linked an account to its identity provider")
		return u, false, nil
	} else if !errors.Is(err, storage.ErrUserNotFound) {
		return nil, false, err
	}
	if !p.DomainAllowed(id) {
		return nil, false, fmt.Errorf("%w: there is no GoHead account for %s; ask an administrator to create one", ErrNoAccount, id.Email)
	}
	role, err := storage.GetRoleByName(p.DefaultRole)
	if err != nil {
		return nil, false, fmt.Errorf("the default role %q of %s sign-in does not exist: %w", p.DefaultRole, p.Label, err)
	}
	username, err := freeUsername(id.Email)
	if err != nil {
		return nil, false, err
	}
	// No password: the account signs in with the provider only (an admin
	// can set one).
	u, err := Build(username, id.Email, sso.RandomString(32), role)
	if err != nil {
		return nil, false, err
	}
	u.SSOProvider, u.SSOSubject = id.Provider, id.Subject
	if err := storage.CreateUser(u); err != nil {
		return nil, false, err
	}
	logger.Log.WithFields(map[string]any{"username": u.Username, "provider": id.Provider, "role": role.Name}).Info("Created an account on sign-in")
	return u, true, nil
}

var notUsername = regexp.MustCompile(`[^a-z0-9._-]+`)

// freeUsername derives an unused username from an email address
// ("jane.doe@x.com" → "jane.doe", then "jane.doe-2"…).
func freeUsername(email string) (string, error) {
	local, _, _ := strings.Cut(strings.ToLower(email), "@")
	base := strings.Trim(notUsername.ReplaceAllString(local, "-"), "-.")
	if base == "" {
		base = "user"
	}
	if len(base) > 40 {
		base = base[:40]
	}
	for i := 1; i <= 100; i++ {
		name := base
		if i > 1 {
			name = fmt.Sprintf("%s-%d", base, i)
		}
		if _, err := storage.GetUserByUsername(name); err != nil {
			return name, nil
		}
	}
	return "", errors.New("could not find a free username")
}
