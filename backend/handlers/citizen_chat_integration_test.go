//go:build integration

package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

func TestCitizenChatJoinUsesServerResolvedGeography(t *testing.T) {
	if config.DB == nil { t.Skip("integration DB unavailable") }
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	owner := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, owner)
	if err := config.DB.Model(&models.SecurityUnit{}).Where("id = ?", unit.ID).
		Updates(map[string]any{"state": "FCT", "lga": "Abuja Municipal", "ward": "Garki"}).Error; err != nil {
		t.Fatalf("set unit place: %v", err)
	}
	ctx, recorder := commJSONContext(t, http.MethodPost, "/community/chat/rooms/join",
		map[string]any{"scope": "lga", "state": "Lagos", "lga": "Ikeja", "locality": "Ikeja"}, owner)
	JoinCitizenChatRoom(ctx)
	if recorder.Code != http.StatusOK { t.Fatalf("want 200, got %d %s", recorder.Code, recorder.Body.String()) }

	var response struct { Room struct { ID uuid.UUID `json:"id"`; State string `json:"state"`; LGA string `json:"lga"`; Locality string `json:"locality"` } `json:"room"` }
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil { t.Fatalf("decode response: %v", err) }
	if response.Room.ID == uuid.Nil { t.Fatal("room ID missing") }
	if response.Room.State != "FCT" || response.Room.LGA != "Abuja Municipal" || response.Room.Locality != "Garki" {
		t.Fatalf("client place names must be ignored; got state=%q lga=%q locality=%q", response.Room.State, response.Room.LGA, response.Room.Locality)
	}
}

func TestCitizenChatCannotReadAcrossLGA(t *testing.T) {
	if config.DB == nil { t.Skip("integration DB unavailable") }
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	first := testutil.MakeUser(t, "unit_admin")
	firstUnit := testutil.MakeUnit(t, first)
	if err := config.DB.Model(&models.SecurityUnit{}).Where("id = ?", firstUnit.ID).
		Updates(map[string]any{"state": "FCT", "lga": "Abuja Municipal", "ward": "Garki"}).Error; err != nil { t.Fatal(err) }

	second := testutil.MakeUser(t, "unit_admin")
	secondUnit := testutil.MakeUnit(t, second)
	if err := config.DB.Model(&models.SecurityUnit{}).Where("id = ?", secondUnit.ID).
		Updates(map[string]any{"state": "FCT", "lga": "Bwari", "ward": "Kubwa"}).Error; err != nil { t.Fatal(err) }

	joinCtx, joinRecorder := commJSONContext(t, http.MethodPost, "/community/chat/rooms/join", map[string]any{"scope": ScopeLGA}, first)
	JoinCitizenChatRoom(joinCtx)
	if joinRecorder.Code != http.StatusOK { t.Fatalf("join: want 200, got %d %s", joinRecorder.Code, joinRecorder.Body.String()) }
	var payload struct { Room struct { ID uuid.UUID `json:"id"` } `json:"room"` }
	if err := json.Unmarshal(joinRecorder.Body.Bytes(), &payload); err != nil { t.Fatal(err) }

	readCtx, readRecorder := commJSONContext(t, http.MethodGet, "/community/chat/rooms/"+payload.Room.ID.String()+"/messages", nil, second)
	readCtx.Params = gin.Params{{Key: "roomId", Value: payload.Room.ID.String()}}
	GetCitizenChatMessages(readCtx)
	if readRecorder.Code != http.StatusForbidden {
		t.Fatalf("cross-LGA read must be denied, got %d %s", readRecorder.Code, readRecorder.Body.String())
	}
}
