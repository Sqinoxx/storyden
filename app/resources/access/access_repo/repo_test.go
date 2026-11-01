package access_repo

import (
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"

	"github.com/Southclaws/storyden/app/resources/access"
	"github.com/Southclaws/storyden/app/resources/post/category"
)

func TestResolveCategoryLocks(t *testing.T) {
	z1, anatomy, histo, z2, open := xid.New(), xid.New(), xid.New(), xid.New(), xid.New()

	parents := map[xid.ID]xid.ID{
		z1:      xid.NilID(),
		anatomy: z1,
		histo:   anatomy,
		z2:      xid.NilID(),
		open:    xid.NilID(),
	}

	z1Pass := access.PassRef{ID: access.PassID(xid.New()), Name: "Z1"}
	bundle := access.PassRef{ID: access.PassID(xid.New()), Name: "Z1-Z3"}

	locks := ResolveCategoryLocks(parents, map[xid.ID][]access.PassRef{
		z1:      {z1Pass, bundle},
		anatomy: {z1Pass},
		z2:      {bundle},
	})

	assert.ElementsMatch(t, []access.PassRef{z1Pass, bundle}, locks[category.CategoryID(z1)])
	assert.ElementsMatch(t, []access.PassRef{z1Pass, bundle}, locks[category.CategoryID(anatomy)], "duplicates from ancestors collapse")
	assert.ElementsMatch(t, []access.PassRef{z1Pass, bundle}, locks[category.CategoryID(histo)], "grandchildren inherit")
	assert.Equal(t, []access.PassRef{bundle}, locks[category.CategoryID(z2)])
	assert.NotContains(t, locks, category.CategoryID(open))
}

func TestResolveCategoryLocksCycle(t *testing.T) {
	a, b := xid.New(), xid.New()
	ref := access.PassRef{ID: access.PassID(xid.New())}

	locks := ResolveCategoryLocks(map[xid.ID]xid.ID{a: b, b: a}, map[xid.ID][]access.PassRef{a: {ref}})

	assert.Equal(t, []access.PassRef{ref}, locks[category.CategoryID(b)])
}

func TestReferenceCode(t *testing.T) {
	cases := map[string]string{
		"Z1-Protokolle":     `^Z1-[A-Z2-9]{4}$`,
		"Physiologie Paket": `^PHYS-[A-Z2-9]{4}$`,
		"  Ärzte":           `^RZTE-[A-Z2-9]{4}$`,
		"€€€":               `^SD-[A-Z2-9]{4}$`,
	}

	for name, pattern := range cases {
		code, err := newReferenceCode(name)
		assert.NoError(t, err)
		assert.Regexp(t, pattern, code, name)
	}
}
