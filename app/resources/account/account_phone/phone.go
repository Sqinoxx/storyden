package account_phone

import (
	"context"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/internal/ent"
	ent_account "github.com/Southclaws/storyden/internal/ent/account"
)

const DefaultCountryCode = "49"

var ErrInvalidPhone = fault.New("invalid phone number", ftag.With(ftag.InvalidArgument))

// Normalise converts user input into E.164 form. Numbers without a country
// prefix are assumed to be in DefaultCountryCode.
func Normalise(raw string) (string, error) {
	var b strings.Builder
	for i, r := range strings.TrimSpace(raw) {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' && i == 0:
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '/' || r == '(' || r == ')' || r == '.':
		default:
			return "", fault.Wrap(ErrInvalidPhone, fmsg.WithDesc("invalid character", "Die Telefonnummer darf nur Ziffern enthalten."))
		}
	}

	s := b.String()
	switch {
	case strings.HasPrefix(s, "+"):
		s = s[1:]
	case strings.HasPrefix(s, "00"):
		s = s[2:]
	case strings.HasPrefix(s, "0"):
		s = DefaultCountryCode + s[1:]
	}

	if len(s) < 8 || len(s) > 15 || s[0] == '0' {
		return "", fault.Wrap(ErrInvalidPhone, fmsg.WithDesc("invalid length", "Bitte gib eine gültige Telefonnummer ein, z. B. +49 151 12345678."))
	}

	return "+" + s, nil
}

type Repository struct {
	db *ent.Client
}

func New(db *ent.Client) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context, id account.AccountID) (opt.Optional[string], error) {
	acc, err := r.db.Account.Query().
		Where(ent_account.ID(xid.ID(id))).
		Select(ent_account.FieldPhone).
		Only(ctx)
	if err != nil {
		return opt.NewEmpty[string](), fault.Wrap(err, fctx.With(ctx))
	}

	return opt.NewPtr(acc.Phone), nil
}

func (r *Repository) GetMany(ctx context.Context, ids ...account.AccountID) (map[account.AccountID]string, error) {
	raw := make([]xid.ID, len(ids))
	for i, id := range ids {
		raw[i] = xid.ID(id)
	}

	accs, err := r.db.Account.Query().
		Where(ent_account.IDIn(raw...), ent_account.PhoneNotNil()).
		Select(ent_account.FieldID, ent_account.FieldPhone).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	out := make(map[account.AccountID]string, len(accs))
	for _, a := range accs {
		out[account.AccountID(a.ID)] = *a.Phone
	}

	return out, nil
}

func (r *Repository) Set(ctx context.Context, id account.AccountID, raw string) (string, error) {
	phone, err := Normalise(raw)
	if err != nil {
		return "", fault.Wrap(err, fctx.With(ctx))
	}

	err = r.db.Account.UpdateOneID(xid.ID(id)).SetPhone(phone).Exec(ctx)
	if err != nil {
		return "", fault.Wrap(err, fctx.With(ctx))
	}

	return phone, nil
}

func (r *Repository) Clear(ctx context.Context, id account.AccountID) error {
	err := r.db.Account.UpdateOneID(xid.ID(id)).ClearPhone().Exec(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	return nil
}
