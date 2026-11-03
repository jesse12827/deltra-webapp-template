package items

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"deltra/webapp/internal/audit"
	"deltra/webapp/internal/dbgen"
)

// ErrNotFound means no item has the requested identifier.
var ErrNotFound = errors.New("item not found")

// ErrConflict means the item changed after the user opened it for editing.
var ErrConflict = errors.New("item was changed by someone else")

// Service runs item business operations.
type Service struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

// NewService returns a Service using pool.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, q: dbgen.New(pool)}
}

// ListParams selects one page of items.
type ListParams struct {
	Search string
	Status string
	Sort   string
	Page   int // 1-based
	Size   int
}

// Page is one page of results plus what the list view needs for pagination.
type Page struct {
	Items      []dbgen.Item
	Total      int64
	Page, Size int
	Pages      int
}

// List returns one page of items matching p. Inputs are normalized here, so
// handlers cannot pass an unbounded page size or an arbitrary sort expression.
func (s *Service) List(ctx context.Context, p ListParams) (Page, error) {
	p.Size = NormalizePageSize(p.Size)
	p.Sort = NormalizeSort(p.Sort)
	if !validStatus(p.Status) {
		p.Status = ""
	}
	search := likeEscape(p.Search)

	total, err := s.q.CountItems(ctx, dbgen.CountItemsParams{Search: search, Status: p.Status})
	if err != nil {
		return Page{}, fmt.Errorf("counting items: %w", err)
	}
	pages := int((total + int64(p.Size) - 1) / int64(p.Size))
	if pages < 1 {
		pages = 1
	}
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Page > pages {
		p.Page = pages
	}
	rows, err := s.q.ListItems(ctx, dbgen.ListItemsParams{
		Search:     search,
		Status:     p.Status,
		Sort:       p.Sort,
		PageLimit:  int32(p.Size),                //nolint:gosec // bounded by NormalizePageSize
		PageOffset: int32((p.Page - 1) * p.Size), //nolint:gosec // bounded by total row count
	})
	if err != nil {
		return Page{}, fmt.Errorf("listing items: %w", err)
	}
	return Page{Items: rows, Total: total, Page: p.Page, Size: p.Size, Pages: pages}, nil
}

// CountByStatus returns the number of items in each status, for the dashboard.
func (s *Service) CountByStatus(ctx context.Context) ([]dbgen.CountItemsByStatusRow, error) {
	return s.q.CountItemsByStatus(ctx)
}

// Get returns one item.
func (s *Service) Get(ctx context.Context, id int64) (dbgen.Item, error) {
	it, err := s.q.GetItem(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return dbgen.Item{}, ErrNotFound
	}
	return it, err
}

// Create validates in and stores a new item with its audit event, in one transaction.
// A non-nil FieldErrors means validation failed and nothing was stored.
func (s *Service) Create(ctx context.Context, userID int64, in Input) (dbgen.Item, FieldErrors, error) {
	c, ferrs := in.Validate()
	if ferrs != nil {
		return dbgen.Item{}, ferrs, nil
	}
	var created dbgen.Item
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		created, err = q.CreateItem(ctx, dbgen.CreateItemParams{
			Title: c.Title, Description: c.Description, Status: c.Status, DueOn: c.DueOn, UserID: userID,
		})
		if err != nil {
			return fmt.Errorf("creating item: %w", err)
		}
		return audit.Record(ctx, q, audit.Event{
			UserID: audit.Ptr(userID), Action: "item.create", RecordType: "item", RecordID: audit.Ptr(created.ID),
			Changes: audit.Diff(nil, fields(created)),
		})
	})
	return created, nil, err
}

// Update validates in and saves it only if the item is still at version, so one
// user's changes never silently overwrite another's.
func (s *Service) Update(ctx context.Context, userID, id int64, version int32, in Input) (dbgen.Item, FieldErrors, error) {
	c, ferrs := in.Validate()
	if ferrs != nil {
		return dbgen.Item{}, ferrs, nil
	}
	var updated dbgen.Item
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		before, err := q.GetItem(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return fmt.Errorf("reading item: %w", err)
		}
		updated, err = q.UpdateItem(ctx, dbgen.UpdateItemParams{
			Title: c.Title, Description: c.Description, Status: c.Status, DueOn: c.DueOn,
			UserID: userID, ID: id, Version: version,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrConflict
		} else if err != nil {
			return fmt.Errorf("updating item: %w", err)
		}
		return audit.Record(ctx, q, audit.Event{
			UserID: audit.Ptr(userID), Action: "item.update", RecordType: "item", RecordID: audit.Ptr(id),
			Changes: audit.Diff(fields(before), fields(updated)),
		})
	})
	return updated, nil, err
}

// fields lists the business fields recorded in audit events.
func fields(it dbgen.Item) map[string]any {
	due := ""
	if it.DueOn != nil {
		due = it.DueOn.Format(time.DateOnly)
	}
	return map[string]any{
		"title": it.Title, "description": it.Description, "status": it.Status, "due_on": due,
	}
}
