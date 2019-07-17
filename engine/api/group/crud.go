package group

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// Create insert a new group in database and set user for given id as group admin.
func Create(db gorp.SqlExecutor, grp sdk.Group, userID int64) (*sdk.Group, error) {
	if err := Insert(db, &grp); err != nil {
		return nil, err
	}

	if err := InsertLinkGroupUser(db, &LinkGroupUser{
		GroupID: grp.ID,
		UserID:  userID,
		Admin:   true,
	}); err != nil {
		return nil, err
	}

	return &grp, nil
}
