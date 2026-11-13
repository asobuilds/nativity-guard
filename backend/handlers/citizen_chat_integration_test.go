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

func TestCitizenChatReportingBlockingAndModeration(t *testing.T) {
	if config.DB == nil { t.Skip("integration DB unavailable") }
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	moderator := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, moderator)
	sender := testutil.MakeUser(t, "unit_admin")
	testutil.MakeMembership(t, sender, unit, models.UnitRoleAdmin, false)
	if err := config.DB.Model(&models.SecurityUnit{}).Where("id = ?", unit.ID).
		Updates(map[string]any{"state": "FCT", "lga": "Abuja Municipal", "ward": "Garki"}).Error; err != nil { t.Fatal(err) }

	join := func(user *models.User) uuid.UUID {
		ctx, recorder := commJSONContext(t, http.MethodPost, "/community/chat/rooms/join", map[string]any{"scope": ScopeLGA}, user)
		JoinCitizenChatRoom(ctx)
		if recorder.Code != http.StatusOK { t.Fatalf("join: %d %s", recorder.Code, recorder.Body.String()) }
		var body struct { Room struct { ID uuid.UUID `json:"id"` } `json:"room"` }
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil { t.Fatal(err) }
		return body.Room.ID
	}
	roomID := join(sender)
	if otherRoom := join(moderator); otherRoom != roomID { t.Fatal("same geographic scope must resolve to the same room") }

	send := func(content string) uuid.UUID {
		ctx, recorder := commJSONContext(t, http.MethodPost, "/community/chat/rooms/"+roomID.String()+"/messages", map[string]any{"content": content}, sender)
		ctx.Params = gin.Params{{Key: "roomId", Value: roomID.String()}}
		SendCitizenChatMessage(ctx)
		if recorder.Code != http.StatusCreated { t.Fatalf("send: %d %s", recorder.Code, recorder.Body.String()) }
		var body struct { Message struct { ID uuid.UUID `json:"id"` } `json:"message"` }
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil { t.Fatal(err) }
		return body.Message.ID
	}

	messageID := send("Please review this message")
	reportCtx, reportRecorder := commJSONContext(t, http.MethodPost, "/community/chat/messages/"+messageID.String()+"/report", map[string]any{"reason": "test abuse report"}, moderator)
	reportCtx.Params = gin.Params{{Key: "messageId", Value: messageID.String()}}
	ReportCitizenChatMessage(reportCtx)
	if reportRecorder.Code != http.StatusCreated { t.Fatalf("report: %d %s", reportRecorder.Code, reportRecorder.Body.String()) }
	var reportBody struct { ReportID uuid.UUID `json:"reportId"` }
	if err := json.Unmarshal(reportRecorder.Body.Bytes(), &reportBody); err != nil { t.Fatal(err) }

	queueCtx, queueRecorder := commJSONContext(t, http.MethodGet, "/community/chat/reports", nil, moderator)
	GetCitizenChatReports(queueCtx)
	if queueRecorder.Code != http.StatusOK { t.Fatalf("moderation queue: %d %s", queueRecorder.Code, queueRecorder.Body.String()) }

	resolveCtx, resolveRecorder := commJSONContext(t, http.MethodPost, "/community/chat/reports/"+reportBody.ReportID.String()+"/resolve", map[string]any{"status": "reviewed", "action": "hide"}, moderator)
	resolveCtx.Params = gin.Params{{Key: "reportId", Value: reportBody.ReportID.String()}}
	ResolveCitizenChatReport(resolveCtx)
	if resolveRecorder.Code != http.StatusOK { t.Fatalf("resolve: %d %s", resolveRecorder.Code, resolveRecorder.Body.String()) }

	secondMessage := send("A second message before block")
	blockCtx, blockRecorder := commJSONContext(t, http.MethodPost, "/community/chat/blocks", map[string]any{"userId": sender.ID.String()}, moderator)
	BlockCitizenChatUser(blockCtx)
	if blockRecorder.Code != http.StatusOK { t.Fatalf("block: %d %s", blockRecorder.Code, blockRecorder.Body.String()) }

	readCtx, readRecorder := commJSONContext(t, http.MethodGet, "/community/chat/rooms/"+roomID.String()+"/messages", nil, moderator)
	readCtx.Params = gin.Params{{Key: "roomId", Value: roomID.String()}}
	GetCitizenChatMessages(readCtx)
	if readRecorder.Code != http.StatusOK { t.Fatalf("read after moderation/block: %d %s", readRecorder.Code, readRecorder.Body.String()) }
	var page struct { Messages []map[string]any `json:"messages"` }
	if err := json.Unmarshal(readRecorder.Body.Bytes(), &page); err != nil { t.Fatal(err) }
	if len(page.Messages) != 0 { t.Fatalf("hidden and blocked sender messages must not appear; got %d (second message %s)", len(page.Messages), secondMessage) }
}
