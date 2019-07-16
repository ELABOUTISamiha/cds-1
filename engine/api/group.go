package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getGroupsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var groups []sdk.Group
		var err error

		withoutDefault := FormBool(r, "withoutDefault")
		if isMaintainer(ctx) {
			groups, err = group.LoadAll(ctx, api.mustDB())
		} else {
			groups, err = group.LoadAllByDeprecatedUserID(ctx, api.mustDB(), getAPIConsumer(ctx).AuthentifiedUser.OldUserStruct.ID)
		}
		if err != nil {
			return err
		}

		// withoutDefault is use by project add, to avoid selecting the default group on project creation
		if withoutDefault {
			var filteredGroups []sdk.Group
			for _, g := range groups {
				if !group.IsDefaultGroupID(g.ID) {
					filteredGroups = append(filteredGroups, g)
				}
			}
			return service.WriteJSON(w, filteredGroups, http.StatusOK)
		}

		return service.WriteJSON(w, groups, http.StatusOK)
	}
}

func (api *API) getGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["permGroupName"]

		g, err := group.LoadByName(ctx, api.mustDB(), name, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, g, http.StatusOK)
	}
}

func (api *API) postGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var data sdk.Group
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		//if err := data.IsValid(); err != nil {
		//	return err
		//}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot begin tx")
		}
		defer tx.Rollback()

		existingGroup, err := group.LoadByName(ctx, tx, data.Name)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if existingGroup != nil {
			return sdk.WithStack(sdk.ErrGroupPresent)
		}
		if err := group.Insert(tx, &data); err != nil {
			return sdk.WrapError(err, "cannot add group")
		}

		consumer := getAPIConsumer(ctx)

		// Add caller into group
		if err := group.InsertUserInGroup(tx, data.ID, consumer.AuthentifiedUser.OldUserStruct.ID, false); err != nil {
			return sdk.WrapError(err, "cannot add user %s in group %s", consumer.AuthentifiedUser.Username, data.Name)
		}
		// and set it admin
		if err := group.SetUserGroupAdmin(tx, data.ID, consumer.AuthentifiedUser.OldUserStruct.ID); err != nil {
			return sdk.WrapError(err, "cannot set user group admin")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit tx")
		}

		return service.WriteJSON(w, data, http.StatusCreated)
	}
}

func (api *API) deleteGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["permGroupName"]
		u := getAPIConsumer(ctx)

		g, err := group.LoadByName(ctx, api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "cannot load %s", name)
		}

		projPerms, err := project.LoadPermissions(api.mustDB(), g.ID)
		if err != nil {
			return sdk.WrapError(err, "cannot load projects for group")
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.DeleteGroupAndDependencies(tx, g); err != nil {
			return sdk.WrapError(err, "cannot delete group")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		groupPerm := sdk.GroupPermission{Group: *g}
		for _, pg := range projPerms {
			event.PublishDeleteProjectPermission(&pg.Project, groupPerm, u)
		}

		return nil
	}
}

func (api *API) updateGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get group name in URL
		vars := mux.Vars(r)
		oldName := vars["permGroupName"]

		var updatedGroup sdk.Group
		if err := service.UnmarshalBody(r, &updatedGroup); err != nil {
			return sdk.WrapError(err, "Cannot unmarshal")
		}

		// TODO
		// Update a group should:
		// - Do not update members
		// - Only rename the group
		// - Update worker model names related to this group
		// - Udpate the action names related to this group

		/*if len(updatedGroup.Admins) == 0 {
			return sdk.WrapError(sdk.ErrGroupNeedAdmin, "Cannot Delete all admins for group %s", updatedGroup.Name)
		}*/

		g, errl := group.LoadByName(ctx, api.mustDB(), oldName)
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load %s", oldName)
		}

		updatedGroup.ID = g.ID
		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.UpdateGroup(tx, &updatedGroup, oldName); err != nil {
			return sdk.WrapError(err, "Cannot update group %s", oldName)
		}

		if err := group.DeleteGroupUserByGroup(tx, &updatedGroup); err != nil {
			return sdk.WrapError(err, "Cannot delete users in group %s", oldName)
		}

		/*for _, a := range updatedGroup.Admins {
			u, err := user.LoadByUsername(ctx, tx, a.Username, user.LoadOptions.WithDeprecatedUser)
			if err != nil {
				return sdk.WrapError(err, "Cannot load user(admins) %s", a.Username)
			}

			if err := group.InsertUserInGroup(tx, updatedGroup.ID, u.OldUserStruct.ID, true); err != nil {
				return sdk.WrapError(err, "Cannot insert admin %s in group %s", a.Username, updatedGroup.Name)
			}
		}*/

		/*for _, a := range updatedGroup.Users {
			u, err := user.LoadByUsername(ctx, tx, a.Username, user.LoadOptions.WithDeprecatedUser)
			if err != nil {
				return sdk.WrapError(err, "Cannot load user(members) %s", a.Username)
			}

			if err := group.InsertUserInGroup(tx, updatedGroup.ID, u.OldUserStruct.ID, false); err != nil {
				return sdk.WrapError(err, "Cannot insert member %s in group %s", a.Username, updatedGroup.Name)
			}
		}*/

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		return service.WriteJSON(w, updatedGroup, http.StatusOK)
	}
}

func (api *API) removeUserFromGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["permGroupName"]
		username := vars["user"]

		db := api.mustDB()

		g, err := group.LoadByName(ctx, db, name)
		if err != nil {
			return sdk.WrapError(err, "cannot load %s", name)
		}

		u, err := user.LoadByUsername(ctx, db, username, user.LoadOptions.WithDeprecatedUser)
		if err != nil {
			return err
		}

		userInGroup, err := group.CheckUserInGroup(db, g.ID, u.OldUserStruct.ID)
		if err != nil {
			return sdk.WrapError(err, "cannot check if user %s is already in the group %s", username, g.Name)
		}
		if !userInGroup {
			return sdk.WrapError(sdk.ErrWrongRequest, "user %s is not in group %s", username, name)
		}

		if err := group.DeleteUserFromGroup(api.mustDB(), g.ID, u.OldUserStruct.ID); err != nil {
			return sdk.WrapError(err, "cannot delete user %s from group %s", username, g.Name)
		}

		return nil
	}
}

func (api *API) addUserInGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get group name in URL
		vars := mux.Vars(r)
		name := vars["permGroupName"]

		var users []string
		if err := service.UnmarshalBody(r, &users); err != nil {
			return sdk.WrapError(err, "Cannot unmarshal")
		}

		db := api.mustDB()

		g, errl := group.LoadByName(ctx, db, name)
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load group %s", name)
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return errb
		}
		defer tx.Rollback()

		for _, username := range users {
			u, err := user.LoadByUsername(ctx, db, username, user.LoadOptions.WithDeprecatedUser)
			if err != nil {
				return err
			}
			userInGroup, errc := group.CheckUserInGroup(api.mustDB(), g.ID, u.OldUserStruct.ID)
			if errc != nil {
				return sdk.WrapError(errc, "Cannot check if user %s is already in the group %s", u.Username, g.Name)
			}
			if !userInGroup {
				if err := group.InsertUserInGroup(api.mustDB(), g.ID, u.OldUserStruct.ID, false); err != nil {
					return sdk.WrapError(err, "Cannot add user %s in group %s", u.Username, g.Name)
				}
			}
		}

		return tx.Commit()
	}
}

func (api *API) setUserGroupAdminHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get group name in URL
		vars := mux.Vars(r)
		name := vars["permGroupName"]
		username := vars["user"]

		db := api.mustDB()

		u, err := user.LoadByUsername(ctx, db, username, user.LoadOptions.WithDeprecatedUser)
		if err != nil {
			return err
		}

		g, errl := group.LoadByName(ctx, db, name)
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load %s", name)
		}

		if err := group.SetUserGroupAdmin(api.mustDB(), g.ID, u.OldUserStruct.ID); err != nil {
			return sdk.WrapError(err, "Cannot set user group admin")
		}

		return nil
	}
}

func (api *API) removeUserGroupAdminHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get group name in URL
		vars := mux.Vars(r)
		name := vars["permGroupName"]
		username := vars["user"]

		db := api.mustDB()

		u, err := user.LoadByUsername(ctx, db, username, user.LoadOptions.WithDeprecatedUser)
		if err != nil {
			return err
		}

		g, errl := group.LoadByName(ctx, db, name)
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load %s", name)
		}

		if err := group.RemoveUserGroupAdmin(db, g.ID, u.OldUserStruct.ID); err != nil {
			return sdk.WrapError(err, "Cannot remove user group admin privilege")
		}

		return nil
	}
}
