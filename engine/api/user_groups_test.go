package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/group"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getUserGroupsHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g1 := assets.InsertGroup(t, db)
	g2 := assets.InsertGroup(t, db)

	u, jwtRaw := assets.InsertLambdaUser(db, g1, g2)
	require.NoError(t, group.SetUserGroupAdmin(context.TODO(), db, g2.ID, u.OldUserStruct.ID))

	uri := api.Router.GetRoute(http.MethodGet, api.getUserGroupsHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var gs []sdk.Group
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &gs))
	require.Equal(t, 2, len(gs))
	assert.Equal(t, g1.Name, gs[0].Name)
	assert.Equal(t, false, gs[0].Admin)
	assert.Equal(t, g2.Name, gs[1].Name)
	assert.Equal(t, true, gs[1].Admin)
}
