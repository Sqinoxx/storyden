package bindings

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_querier"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/services/authentication/session"
)

var errPrivilegeEscalation = fault.New("privilege escalation", ftag.With(ftag.PermissionDenied))

func requireGrantable(ctx context.Context, perms rbac.PermissionList) error {
	caller, err := session.GetPermissions(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if caller.HasAny(rbac.PermissionAdministrator) || caller.HasAll(perms...) {
		return nil
	}

	return fault.Wrap(errPrivilegeEscalation, fctx.With(ctx),
		fmsg.WithDesc("beyond own permissions", "You cannot grant or change permissions that you do not hold yourself."))
}

func requireManageableAccount(ctx context.Context, accounts *account_querier.Querier, targetID account.AccountID) error {
	caller, err := session.GetPermissions(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if caller.HasAny(rbac.PermissionAdministrator) {
		return nil
	}

	if callerID, ok := session.GetOptAccountID(ctx).Get(); ok && callerID == targetID {
		return nil
	}

	target, err := accounts.GetByID(ctx, targetID)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if target.Roles.Permissions().HasAny(rbac.PermissionAdministrator) {
		return fault.Wrap(errPrivilegeEscalation, fctx.With(ctx),
			fmsg.WithDesc("administrator account", "Only administrators can manage administrator accounts."))
	}

	return nil
}
