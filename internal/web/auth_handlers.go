package web

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"deltra/webapp/internal/audit"
	"deltra/webapp/internal/auth"
	"deltra/webapp/internal/config"
	"deltra/webapp/internal/dbgen"
)

type signinView struct {
	DevMode bool
	Users   []dbgen.User // development sign-in only: synthetic users to choose from
}

func (s *server) signinPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UserFrom(r.Context()); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	v := signinView{DevMode: s.Config.AuthMode == config.AuthDev}
	if v.DevMode {
		users, err := s.q.ListActiveUsers(r.Context())
		if err != nil {
			s.serverError(w, r, err)
			return
		}
		v.Users = users
	}
	s.render.page(w, r, http.StatusOK, "signin", pageData{Title: "Sign in", Data: v})
}

// devSignin signs in as a synthetic user. It is registered only when
// AUTH_MODE=dev, which configuration refuses in production.
func (s *server) devSignin(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PostFormValue("user_id"), 10, 64)
	if err != nil {
		s.denySignin(w, r, nil, "invalid user")
		return
	}
	u, err := s.q.GetUser(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		s.denySignin(w, r, nil, "unknown user")
		return
	} else if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.completeSignin(w, r, u)
}

func (s *server) entraBegin(w http.ResponseWriter, r *http.Request) {
	dest, err := s.Entra.Begin(w)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, dest, http.StatusFound)
}

func (s *server) entraCallback(w http.ResponseWriter, r *http.Request) {
	email, err := s.Entra.Finish(w, r)
	if err != nil {
		s.Logger.WarnContext(r.Context(), "Entra ID sign-in failed", "err", err)
		s.denySignin(w, r, nil, "sign-in failed")
		return
	}
	u, err := s.q.GetUserByEmail(r.Context(), email)
	if errors.Is(err, pgx.ErrNoRows) {
		s.denySignin(w, r, nil, "no application user for "+email)
		return
	} else if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.completeSignin(w, r, u)
}

// completeSignin admits an active user: a new session, a new cookie, and an audit event.
func (s *server) completeSignin(w http.ResponseWriter, r *http.Request, u dbgen.User) {
	if !u.Active {
		s.denySignin(w, r, &u.ID, "account inactive")
		return
	}
	token, expires, err := s.Sessions.Create(r.Context(), u.ID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := audit.Record(r.Context(), s.q, audit.Event{UserID: audit.Ptr(u.ID), Action: "signin.success"}); err != nil {
		s.serverError(w, r, err)
		return
	}
	auth.SetCookie(w, token, expires)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// denySignin refuses access with a clear message and records why.
func (s *server) denySignin(w http.ResponseWriter, r *http.Request, userID *int64, reason string) {
	if err := audit.Record(r.Context(), s.q, audit.Event{
		UserID: userID, Action: "signin.denied", Detail: map[string]string{"reason": reason},
	}); err != nil {
		s.Logger.ErrorContext(r.Context(), "recording sign-in denial", "err", err)
	}
	s.render.page(w, r, http.StatusForbidden, "error", pageData{
		Title: "Access denied",
		Data: errorView{
			Message: "You are not set up to use this application, or your account is inactive. Contact an administrator.",
		},
	})
}

func (s *server) signout(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	if c, err := r.Cookie(auth.CookieName); err == nil {
		if err := s.Sessions.Delete(r.Context(), c.Value); err != nil {
			s.serverError(w, r, err)
			return
		}
	}
	if err := audit.Record(r.Context(), s.q, audit.Event{UserID: audit.Ptr(u.ID), Action: "signout"}); err != nil {
		s.Logger.ErrorContext(r.Context(), "recording sign-out", "err", err)
	}
	auth.ClearCookie(w)
	http.Redirect(w, r, "/signin", http.StatusSeeOther)
}
