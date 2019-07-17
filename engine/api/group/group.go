package group

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// DeleteGroupAndDependencies deletes group and all subsequent group_project, pipeline_project
func DeleteGroupAndDependencies(db gorp.SqlExecutor, group *sdk.Group) error {
	if err := DeleteGroupUserByGroup(db, group); err != nil {
		return sdk.WrapError(err, "deleteGroupAndDependencies: Cannot delete group user %s", group.Name)
	}

	if err := deleteGroupProjectByGroup(db, group); err != nil {
		return sdk.WrapError(err, "deleteGroupAndDependencies: Cannot delete group project %s", group.Name)
	}

	if err := deleteGroup(db, group); err != nil {
		return sdk.WrapError(err, "deleteGroupAndDependencies: Cannot delete group %s", group.Name)
	}

	// TODO EVENT Send event for all dependencies

	return nil
}

// DeleteUserFromGroup remove user from group
func DeleteUserFromGroup(db gorp.SqlExecutor, groupID, userID int64) error {
	// Check if there are admin left
	var isAdm bool
	err := db.QueryRow(`SELECT group_admin FROM "group_user" WHERE group_id = $1 AND user_id = $2`, groupID, userID).Scan(&isAdm)
	if err != nil {
		return err
	}

	if isAdm {
		var nbAdm int
		err = db.QueryRow(`SELECT COUNT(id) FROM "group_user" WHERE group_id = $1 AND group_admin = true`, groupID).Scan(&nbAdm)
		if err != nil {
			return err
		}

		if nbAdm <= 1 {
			return sdk.ErrNotEnoughAdmin
		}
	}

	query := `DELETE FROM group_user WHERE group_id=$1 AND user_id=$2`
	_, err = db.Exec(query, groupID, userID)
	return err
}

// CheckUserInDefaultGroup insert user in default group
func CheckUserInDefaultGroup(ctx context.Context, db gorp.SqlExecutor, userID int64) error {
	if DefaultGroup == nil || DefaultGroup.ID == 0 {
		return nil
	}

	inGroup, err := CheckUserInGroup(ctx, db, DefaultGroup.ID, userID)
	if err != nil {
		return err
	}
	if !inGroup {
		return InsertLinkGroupUser(db, &LinkGroupUser{
			GroupID: DefaultGroup.ID,
			UserID:  userID,
			Admin:   false,
		})
	}

	return nil
}

// DeleteGroupUserByGroup Delete all user from a group
func DeleteGroupUserByGroup(db gorp.SqlExecutor, group *sdk.Group) error {
	query := `DELETE FROM group_user WHERE group_id=$1`
	_, err := db.Exec(query, group.ID)
	return err
}

// UpdateGroup updates group informations in database
func UpdateGroup(db gorp.SqlExecutor, g *sdk.Group, oldName string) error {
	rx := sdk.NamePatternRegex
	if !rx.MatchString(g.Name) {
		return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid group name. It should match %s", sdk.NamePattern))
	}

	query := `UPDATE "group" set name=$1 WHERE name=$2`
	_, err := db.Exec(query, g.Name, oldName)

	if err != nil && strings.Contains(err.Error(), "idx_group_name") {
		return sdk.ErrGroupExists
	}

	return err
}

// LoadGroupByProject retrieves all groups related to project
func LoadGroupByProject(db gorp.SqlExecutor, project *sdk.Project) error {
	query := `
    SELECT "group".id, "group".name, project_group.role
    FROM "group"
	  JOIN project_group ON project_group.group_id = "group".id
    WHERE project_group.project_id = $1
    ORDER BY "group".name ASC
  `
	rows, err := db.Query(query, project.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var group sdk.Group
		var perm int
		if err := rows.Scan(&group.ID, &group.Name, &perm); err != nil {
			return err
		}
		project.ProjectGroups = append(project.ProjectGroups, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return nil
}

func deleteGroup(db gorp.SqlExecutor, g *sdk.Group) error {
	query := `DELETE FROM "group" WHERE id=$1`
	_, err := db.Exec(query, g.ID)
	return err
}

// RemoveUserGroupAdmin remove the privilege to perform operations on given group
func RemoveUserGroupAdmin(db gorp.SqlExecutor, groupID int64, userID int64) error {
	query := `UPDATE "group_user" SET group_admin = false WHERE group_id = $1 AND user_id = $2`
	_, err := db.Exec(query, groupID, userID)
	return err
}
