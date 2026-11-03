package web

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"deltra/webapp/internal/auth"
	"deltra/webapp/internal/dbgen"
	"deltra/webapp/internal/items"
)

// maxFormBytes bounds request bodies for ordinary forms.
const maxFormBytes = 1 << 20

type dashboardView struct {
	Counts []dbgen.CountItemsByStatusRow
}

func (s *server) dashboard(w http.ResponseWriter, r *http.Request) {
	counts, err := s.Items.CountByStatus(r.Context())
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.render.page(w, r, http.StatusOK, "dashboard", pageData{Title: "Dashboard", Nav: "dashboard", Data: dashboardView{Counts: counts}})
}

type itemsListView struct {
	Page      items.Page
	Q, Status string
	Sort      string
	Statuses  []string
	PageSizes []int
	PrevURL   string
	NextURL   string
	SortTitle string // URL that sorts by title (toggling direction)
	SortDue   string // URL that sorts by due date (toggling direction)
}

func (s *server) itemsList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	size, _ := strconv.Atoi(q.Get("size"))
	p, err := s.Items.List(r.Context(), items.ListParams{
		Search: q.Get("q"), Status: q.Get("status"), Sort: q.Get("sort"), Page: page, Size: size,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	v := itemsListView{
		Page: p, Q: q.Get("q"), Status: q.Get("status"), Sort: items.NormalizeSort(q.Get("sort")),
		Statuses: items.Statuses, PageSizes: items.PageSizes,
	}
	link := func(sort string, page int) string { return listURL(v.Q, v.Status, sort, page, p.Size) }
	if p.Page > 1 {
		v.PrevURL = link(v.Sort, p.Page-1)
	}
	if p.Page < p.Pages {
		v.NextURL = link(v.Sort, p.Page+1)
	}
	v.SortTitle = link(toggle(v.Sort, "title"), 1)
	v.SortDue = link(toggle(v.Sort, "due"), 1)

	d := pageData{Title: "Items", Nav: "items", Data: v}
	if isHTMX(r, "results") {
		s.render.block(w, r, http.StatusOK, "items_list", "results", d)
		return
	}
	s.render.page(w, r, http.StatusOK, "items_list", d)
}

// listURL builds a list-page URL that preserves search, filter, sort, and paging,
// so the page can be reloaded or bookmarked.
func listURL(q, status, sort string, page, size int) string {
	v := url.Values{}
	for k, val := range map[string]string{"q": q, "status": status, "sort": sort} {
		if val != "" {
			v.Set(k, val)
		}
	}
	if page > 1 {
		v.Set("page", strconv.Itoa(page))
	}
	if size != items.PageSizes[0] {
		v.Set("size", strconv.Itoa(size))
	}
	if len(v) == 0 {
		return "/items"
	}
	return "/items?" + v.Encode()
}

func toggle(current, key string) string {
	if current == key {
		return "-" + key
	}
	return key
}

type itemFormView struct {
	ID       int64
	Version  int32
	Input    items.Input
	Errors   items.FieldErrors
	Statuses []string
	Conflict bool
}

type itemDetailView struct {
	Item    dbgen.Item
	History []dbgen.AuditEvent
}

func (s *server) itemNew(w http.ResponseWriter, r *http.Request) {
	s.render.page(w, r, http.StatusOK, "item_form", pageData{
		Title: "New item", Nav: "items",
		Data: itemFormView{Input: items.Input{Status: "open"}, Statuses: items.Statuses},
	})
}

func (s *server) itemCreate(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	in := readItemForm(w, r)
	it, ferrs, err := s.Items.Create(r.Context(), u.ID, in)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if ferrs != nil {
		// Re-show the form with the user's input and an error beside each field.
		s.render.page(w, r, http.StatusUnprocessableEntity, "item_form", pageData{
			Title: "New item", Nav: "items",
			Data: itemFormView{Input: in, Errors: ferrs, Statuses: items.Statuses},
		})
		return
	}
	http.Redirect(w, r, "/items/"+strconv.FormatInt(it.ID, 10), http.StatusSeeOther)
}

func (s *server) itemDetail(w http.ResponseWriter, r *http.Request) {
	it, ok := s.loadItem(w, r)
	if !ok {
		return
	}
	history, err := s.q.ListAuditEventsForRecord(r.Context(), dbgen.ListAuditEventsForRecordParams{
		RecordType: "item", RecordID: &it.ID,
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.render.page(w, r, http.StatusOK, "item_detail", pageData{
		Title: it.Title, Nav: "items", Data: itemDetailView{Item: it, History: history},
	})
}

func (s *server) itemEdit(w http.ResponseWriter, r *http.Request) {
	it, ok := s.loadItem(w, r)
	if !ok {
		return
	}
	s.render.page(w, r, http.StatusOK, "item_form", pageData{
		Title: "Edit item", Nav: "items",
		Data: itemFormView{ID: it.ID, Version: it.Version, Statuses: items.Statuses, Input: items.Input{
			Title: it.Title, Description: it.Description, Status: it.Status, DueOn: dateString(it.DueOn),
		}},
	})
}

func (s *server) itemUpdate(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.notFound(w, r)
		return
	}
	in := readItemForm(w, r)
	version, err := strconv.ParseInt(r.PostFormValue("version"), 10, 32)
	if err != nil {
		s.render.page(w, r, http.StatusBadRequest, "error", pageData{Title: "Invalid request", Data: errorView{Message: "The form was incomplete. Reload the page and try again."}})
		return
	}
	v32 := int32(version) //nolint:gosec // ParseInt with bitSize 32 guarantees the range
	_, ferrs, err := s.Items.Update(r.Context(), u.ID, id, v32, in)
	view := itemFormView{ID: id, Version: v32, Input: in, Errors: ferrs, Statuses: items.Statuses}
	switch {
	case errors.Is(err, items.ErrNotFound):
		s.notFound(w, r)
	case errors.Is(err, items.ErrConflict):
		view.Conflict = true
		s.render.page(w, r, http.StatusConflict, "item_form", pageData{Title: "Edit item", Nav: "items", Data: view})
	case err != nil:
		s.serverError(w, r, err)
	case ferrs != nil:
		s.render.page(w, r, http.StatusUnprocessableEntity, "item_form", pageData{Title: "Edit item", Nav: "items", Data: view})
	default:
		http.Redirect(w, r, "/items/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
	}
}

func (s *server) loadItem(w http.ResponseWriter, r *http.Request) (dbgen.Item, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.notFound(w, r)
		return dbgen.Item{}, false
	}
	it, err := s.Items.Get(r.Context(), id)
	if errors.Is(err, items.ErrNotFound) {
		s.notFound(w, r)
		return dbgen.Item{}, false
	} else if err != nil {
		s.serverError(w, r, err)
		return dbgen.Item{}, false
	}
	return it, true
}

func readItemForm(w http.ResponseWriter, r *http.Request) items.Input {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	return items.Input{
		Title:       r.PostFormValue("title"),
		Description: r.PostFormValue("description"),
		Status:      r.PostFormValue("status"),
		DueOn:       r.PostFormValue("due_on"),
	}
}

func dateString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.DateOnly)
}

type auditView struct {
	Events []dbgen.AuditEvent
}

func (s *server) auditList(w http.ResponseWriter, r *http.Request) {
	events, err := s.q.ListAuditEvents(r.Context(), 200)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.render.page(w, r, http.StatusOK, "audit", pageData{Title: "Audit log", Nav: "admin", Data: auditView{Events: events}})
}
