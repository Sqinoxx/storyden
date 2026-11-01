package access

import (
	"testing"
	"time"

	"github.com/Southclaws/opt"
	"github.com/stretchr/testify/assert"
)

func TestGrantIsEffective(t *testing.T) {
	now := time.Now()

	assert.True(t, (&Grant{Status: GrantStatusActive}).IsEffective(now))
	assert.True(t, (&Grant{Status: GrantStatusActive, ExpiresAt: opt.New(now.Add(time.Minute))}).IsEffective(now))
	assert.False(t, (&Grant{Status: GrantStatusActive, ExpiresAt: opt.New(now)}).IsEffective(now))
	assert.False(t, (&Grant{Status: GrantStatusActive, ExpiresAt: opt.New(now.Add(-time.Minute))}).IsEffective(now))
	assert.False(t, (&Grant{Status: GrantStatusRequested}).IsEffective(now))
	assert.False(t, (&Grant{Status: GrantStatusRevoked}).IsEffective(now))
	assert.False(t, (&Grant{Status: GrantStatusRejected}).IsEffective(now))
}
