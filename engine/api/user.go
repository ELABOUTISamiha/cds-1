package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// DeleteUserHandler removes a user
func (api *API) deleteUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["username"]

		if !deprecatedGetUser(ctx).Admin && username != deprecatedGetUser(ctx).Username {
			return service.WriteJSON(w, nil, http.StatusForbidden)
		}

		usr, err := user.LoadUserByUsername(api.mustDB(), username)
		if err != nil {
			return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Cannot load user %s", username)
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "Cannot start transaction")
		}
		defer tx.Rollback()

		if err := user.Delete(tx, usr.ID); err != nil {
			return sdk.WrapError(err, "cannot delete user")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		return nil
	}
}

// GetUserHandler returns a specific user's information
func (api *API) getUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["username"]

		u := getAPIConsumer(ctx)

		if !u.Admin() && username != u.Username {
			return service.WriteJSON(w, nil, http.StatusForbidden)
		}

		if err := loadUserPermissions(api.mustDB(), api.Cache, u); err != nil {
			return sdk.WrapError(err, "getUserHandler: Cannot get user group and project from db")
		}

		return service.WriteJSON(w, u, http.StatusOK)
	}
}

// UpdateUserHandler modifies user informations
func (api *API) updateUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//vars := mux.Vars(r)
		//username := vars["username"]
		//
		//if !deprecatedGetUser(ctx).Admin && username != deprecatedGetUser(ctx).Username {
		//	return service.WriteJSON(w, nil, http.StatusForbidden)
		//}
		//
		//usr, err := user.LoadUserByUsername(api.mustDB(), username)
		//if err != nil {
		//	return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Cannot load user %s", username)
		//}
		//
		//u, err := user.GetDeprecatedUser(api.mustDB(), usr)
		//if err != nil {
		//	return err
		//}
		//
		//var userBody sdk.User
		//if err := service.UnmarshalBody(r, &userBody); err != nil {
		//	return err
		//}
		//
		//userBody.ID = userDB.ID
		//
		//if !user.IsValidEmail(userBody.Email) {
		//	return sdk.WrapError(sdk.ErrWrongRequest, "updateUserHandler: Email address %s is not valid", userBody.Email)
		//}
		//
		//if err := user.UpdateUser(api.mustDB(), userBody); err != nil {
		//	return sdk.WrapError(err, "updateUserHandler: Cannot update user table")
		//}
		//
		//return service.WriteJSON(w, userBody, http.StatusOK)
		return nil
	}
}

// GetUsers fetches all users from databases
func (api *API) getUsersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		users, err := user.LoadAll(api.mustDB(), user.LoadOptions.WithContacts)
		if err != nil {
			return sdk.WrapError(err, "GetUsers: Cannot load user from db")
		}
		return service.WriteJSON(w, users, http.StatusOK)
	}
}

// getUserLoggedHandler check if the current user is connected
func (api *API) getUserLoggedHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		h := r.Header.Get(sdk.SessionTokenHeader)
		if h == "" {
			return sdk.ErrUnauthorized
		}

		key := sessionstore.SessionKey(h)
		if ok, _ := auth.Store.Exists(key); !ok {
			return sdk.ErrUnauthorized
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

//AuthModeHandler returns the auth mode : local ok ldap
/* func (api *API) authModeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		mode := "local"
		if _, ldap := api.Router.AuthDriver.(*auth.LDAPClient); ldap {
			mode = "ldap"
		}
		res := map[string]string{
			"auth_mode": mode,
		}
		return service.WriteJSON(w, res, http.StatusOK)
	}
} */
