package web

import (
	"errors"
	"net/http"
	"strconv"
	"time"

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
// AUTH_MODE=dev, which configuration permits only when APP_ENV=development.
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
	id, err := s.Entra.Finish(w, r)
	if err != nil {
		s.Logger.WarnContext(r.Context(), "Entra ID sign-in failed", "err", err)
		s.denySignin(w, r, nil, "sign-in failed")
		return
	}
	// Match on the tenant ID and object ID only, never on email
	// (Stack Section 10.1).
	u, err := auth.FindUser(r.Context(), s.q, id)
	if errors.Is(err, pgx.ErrNoRows) {
		s.denySignin(w, r, nil, unlinkedReason(id))
		return
	} else if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.completeSignin(w, r, u)
}

// unlinkedReason is the signin.denied reason for an Entra ID account that no
// application user is linked to. It names the tenant and object IDs, which an
// administrator needs to link the account, and never the email claim, which
// the directory controls (Stack Section 10.1).
func unlinkedReason(id auth.Identity) string {
	return "no application user for Entra ID tenant " + id.TenantID + ", object " + id.ObjectID
}

// completeSignin admits an active user. In one transaction it ends any session
// the browser already had, starts a new one, and records the audit event, so a
// failed audit write leaves no session behind (Stack Sections 10.2 and 12).
func (s *server) completeSignin(w http.ResponseWriter, r *http.Request, u dbgen.User) {
	if !u.Active {
		s.denySignin(w, r, &u.ID, "account inactive")
		return
	}
	ctx := r.Context()
	var (
		token   string
		expires time.Time
	)
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		sessions := s.Sessions.WithTx(tx)
		if c, err := r.Cookie(auth.CookieName); err == nil {
			if err := sessions.Delete(ctx, c.Value); err != nil {
				return err
			}
		}
		var err error
		if token, expires, err = sessions.Create(ctx, u.ID); err != nil {
			return err
		}
		return audit.Record(ctx, s.q.WithTx(tx), audit.Event{UserID: audit.Ptr(u.ID), Action: "signin.success"})
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	auth.SetCookie(w, token, expires)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// denySignin refuses access with a clear message and records why. If the
// denial cannot be recorded, the request fails; sign-in is refused either way.
func (s *server) denySignin(w http.ResponseWriter, r *http.Request, userID *int64, reason string) {
	if err := audit.Record(r.Context(), s.q, audit.Event{
		UserID: userID, Action: "signin.denied", Detail: map[string]string{"reason": reason},
	}); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.render.page(w, r, http.StatusForbidden, "error", pageData{
		Title: "Access denied",
		Data: errorView{
			Message: "You are not set up to use this application, or your account is inactive. Contact an administrator.",
		},
	})
}

// signout deletes the session and records the event in one transaction. If the
// event cannot be recorded, the request fails and the session is unchanged.
func (s *server) signout(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	ctx := r.Context()
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		if c, err := r.Cookie(auth.CookieName); err == nil {
			if err := s.Sessions.WithTx(tx).Delete(ctx, c.Value); err != nil {
				return err
			}
		}
		return audit.Record(ctx, s.q.WithTx(tx), audit.Event{UserID: audit.Ptr(u.ID), Action: "signout"})
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	auth.ClearCookie(w)
	http.Redirect(w, r, "/signin", http.StatusSeeOther)
}
