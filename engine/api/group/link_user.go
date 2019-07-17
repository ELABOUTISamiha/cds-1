package group

import (
	"context"

	"github.com/ovh/cds/sdk"

	"github.com/go-gorp/gorp"
)

// SetUserGroupAdmin allows a user to perform operations on given group
func SetUserGroupAdmin(ctx context.Context, db gorp.SqlExecutor, groupID int64, userID int64) error {
	l, err := LoadLinkGroupUserForGroupIDAndUserID(ctx, db, groupID, userID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}
	if l == nil {
		return sdk.NewErrorFrom(sdk.ErrForbidden, "given user %d is not member of group %d", userID, groupID)
	}

	if l.Admin {
		return nil
	}
	l.Admin = true

	return sdk.WrapError(UpdateLinkGroupUser(db, l), "cannot set user %d group admin of %d", userID, groupID)
}

// CheckUserInGroup verivies that user is in given group
func CheckUserInGroup(ctx context.Context, db gorp.SqlExecutor, groupID, userID int64) (bool, error) {
	l, err := LoadLinkGroupUserForGroupIDAndUserID(ctx, db, groupID, userID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return false, err
	}
	return l == nil, nil
}
