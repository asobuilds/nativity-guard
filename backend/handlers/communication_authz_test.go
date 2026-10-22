//go:build integration

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

func commJSONContext(t *testing.T, method, target string, body map[string]any, user *models.User) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	payload, _ := json.Marshal(body)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	if user != nil {
		ctx.Set("user", user)
	}
	return ctx, recorder
}

// makeRoom creates a room inside a real unit so the FK constraints hold.
func makeRoom(t *testing.T, unit *models.SecurityUnit, creator *models.User) *models.CommunicationRoom {
	t.Helper()
	room := &models.CommunicationRoom{
		UnitID:    unit.ID,
		Name:      "Ops Room " + uuid.NewString()[:8],
		Type:      "unit",
		Status:    "active",
		CreatedBy: creator.ID,
	}
	if err := config.DB.Create(room).Error; err != nil {
		t.Fatalf("makeRoom: %v", err)
	}
	return room
}

func addRoomMember(t *testing.T, roomID, userID uuid.UUID, role, status string) {
	t.Helper()
	now := time.Now()
	m := &models.CommunicationRoomMember{
		RoomID:   roomID,
		UserID:   userID,
		Role:     role,
		Status:   status,
		JoinedAt: &now,
	}
	if err := config.DB.Create(m).Error; err != nil {
		t.Fatalf("addRoomMember: %v", err)
	}
}

// setUserUnit persists a unit on the user's row so unit-scoped lookups
// performed against the database resolve correctly.
func setUserUnit(t *testing.T, user *models.User, unitID uuid.UUID) {
	t.Helper()
	user.UnitID = &unitID
	if err := config.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("unit_id", unitID).Error; err != nil {
		t.Fatalf("setUserUnit: %v", err)
	}
}

func TestCommunicationCreateRoomRejectsCitizen(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)

	citizen := testutil.MakeUser(t, "citizen")
	ctx, recorder := commJSONContext(t, "POST", "/communication/rooms", map[string]any{
		"unitId": unit.ID.String(), "name": "Sneaky Room",
	}, citizen)

	CreateRoom(ctx)

	if recorder.Code != 403 {
		t.Fatalf("want 403 for citizen room creation, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationCreateRoomAddsCreatorAsMember(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	setUserUnit(t, head, unit.ID)

	ctx, recorder := commJSONContext(t, "POST", "/communication/rooms", map[string]any{
		"unitId": unit.ID.String(), "name": "Ops Room",
	}, head)

	CreateRoom(ctx)

	if recorder.Code != 201 {
		t.Fatalf("want 201 for admin room creation, got %d %s", recorder.Code, recorder.Body.String())
	}

	var count int64
	config.DB.Model(&models.CommunicationRoomMember{}).
		Where("user_id = ? AND role = ? AND status = ?", head.ID, "admin", "active").
		Count(&count)
	if count != 1 {
		t.Fatalf("want creator enrolled as active admin member, got %d rows", count)
	}
}

func TestCommunicationGetRoomsRejectsCrossUnitAdmin(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	own := testutil.MakeUnit(t, head)
	other := testutil.MakeUnit(t, testutil.MakeUser(t, "unit_admin"))

	setUserUnit(t, head, own.ID)
	ctx, recorder := commJSONContext(t, "GET", "/communication/units/"+other.ID.String()+"/rooms", nil, head)
	ctx.Params = gin.Params{{Key: "id", Value: other.ID.String()}}

	GetRooms(ctx)

	if recorder.Code != 403 {
		t.Fatalf("want 403 for cross-unit room listing, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationGetRoomsAllowsOwnUnit(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	own := testutil.MakeUnit(t, head)
	setUserUnit(t, head, own.ID)

	ctx, recorder := commJSONContext(t, "GET", "/communication/units/"+own.ID.String()+"/rooms", nil, head)
	ctx.Params = gin.Params{{Key: "id", Value: own.ID.String()}}

	GetRooms(ctx)

	if recorder.Code != 200 {
		t.Fatalf("want 200 for own-unit room listing, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationSendMessageRejectsNonMember(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	outsider := testutil.MakeUser(t, "citizen")
	ctx, recorder := commJSONContext(t, "POST", "/communication/messages", map[string]any{
		"roomId": room.ID.String(), "message": "let me in",
	}, outsider)

	SendMessage(ctx)

	if recorder.Code != 403 {
		t.Fatalf("want 403 for non-member message, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationSendMessageAllowsActiveMember(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	member := testutil.MakeUser(t, "citizen")
	addRoomMember(t, room.ID, member.ID, "member", "active")

	ctx, recorder := commJSONContext(t, "POST", "/communication/messages", map[string]any{
		"roomId": room.ID.String(), "message": "hello team",
	}, member)

	SendMessage(ctx)

	if recorder.Code != 201 {
		t.Fatalf("want 201 for active member message, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationSendMessageRejectsRemovedMember(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	former := testutil.MakeUser(t, "citizen")
	addRoomMember(t, room.ID, former.ID, "member", "removed")

	ctx, recorder := commJSONContext(t, "POST", "/communication/messages", map[string]any{
		"roomId": room.ID.String(), "message": "still here?",
	}, former)

	SendMessage(ctx)

	if recorder.Code != 403 {
		t.Fatalf("want 403 for removed member message, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationGetMessagesRejectsNonMember(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	outsider := testutil.MakeUser(t, "citizen")
	ctx, recorder := commJSONContext(t, "GET", "/communication/rooms/"+room.ID.String()+"/messages", nil, outsider)
	ctx.Params = gin.Params{{Key: "roomId", Value: room.ID.String()}}

	GetMessages(ctx)

	if recorder.Code != 403 {
		t.Fatalf("want 403 for non-member message read, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationGetMessagesAllowsActiveMember(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	member := testutil.MakeUser(t, "citizen")
	addRoomMember(t, room.ID, member.ID, "member", "active")

	ctx, recorder := commJSONContext(t, "GET", "/communication/rooms/"+room.ID.String()+"/messages", nil, member)
	ctx.Params = gin.Params{{Key: "roomId", Value: room.ID.String()}}

	GetMessages(ctx)

	if recorder.Code != 200 {
		t.Fatalf("want 200 for active member message read, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationInitiateCallRejectsNonMember(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	outsider := testutil.MakeUser(t, "citizen")
	ctx, recorder := commJSONContext(t, "POST", "/communication/calls", map[string]any{
		"roomId": room.ID.String(), "callType": "group",
	}, outsider)

	InitiateCall(ctx)

	if recorder.Code != 403 {
		t.Fatalf("want 403 for non-member call, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationEndCallRejectsNonInitiator(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	caller := testutil.MakeUser(t, "citizen")
	addRoomMember(t, room.ID, caller.ID, "member", "active")

	now := time.Now()
	call := &models.VoiceCall{
		RoomID:    room.ID,
		CallerID:  caller.ID,
		CallType:  "group",
		Status:    "initiated",
		StartedAt: &now,
	}
	if err := config.DB.Create(call).Error; err != nil {
		t.Fatalf("create call: %v", err)
	}

	other := testutil.MakeUser(t, "citizen")
	addRoomMember(t, room.ID, other.ID, "member", "active")
	ctx, recorder := commJSONContext(t, "PUT", "/communication/calls/"+call.ID.String()+"/end", map[string]any{
		"quality": "good",
	}, other)
	ctx.Params = gin.Params{{Key: "id", Value: call.ID.String()}}

	EndCall(ctx)

	if recorder.Code != 403 {
		t.Fatalf("want 403 for non-initiator ending call, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationEndCallAllowsInitiator(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	caller := testutil.MakeUser(t, "citizen")
	addRoomMember(t, room.ID, caller.ID, "member", "active")

	now := time.Now()
	call := &models.VoiceCall{
		RoomID:    room.ID,
		CallerID:  caller.ID,
		CallType:  "group",
		Status:    "initiated",
		StartedAt: &now,
	}
	if err := config.DB.Create(call).Error; err != nil {
		t.Fatalf("create call: %v", err)
	}

	ctx, recorder := commJSONContext(t, "PUT", "/communication/calls/"+call.ID.String()+"/end", map[string]any{
		"quality": "good",
	}, caller)
	ctx.Params = gin.Params{{Key: "id", Value: call.ID.String()}}

	EndCall(ctx)

	if recorder.Code != 200 {
		t.Fatalf("want 200 for initiator ending call, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationAddRoomMemberRejectsCitizen(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	candidate := testutil.MakeUser(t, "citizen")
	setUserUnit(t, candidate, unit.ID)

	citizen := testutil.MakeUser(t, "citizen")
	ctx, recorder := commJSONContext(t, "POST", "/communication/rooms/"+room.ID.String()+"/members", map[string]any{
		"userId": candidate.ID.String(),
	}, citizen)
	ctx.Params = gin.Params{{Key: "roomId", Value: room.ID.String()}}

	AddRoomMember(ctx)

	if recorder.Code != 403 {
		t.Fatalf("want 403 for citizen adding member, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationAddRoomMemberRejectsCrossUnitUser(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	otherUnit := testutil.MakeUnit(t, testutil.MakeUser(t, "unit_admin"))
	room := makeRoom(t, unit, head)

	outsider := testutil.MakeUser(t, "citizen")
	setUserUnit(t, outsider, otherUnit.ID)

	setUserUnit(t, head, unit.ID)
	ctx, recorder := commJSONContext(t, "POST", "/communication/rooms/"+room.ID.String()+"/members", map[string]any{
		"userId": outsider.ID.String(),
	}, head)
	ctx.Params = gin.Params{{Key: "roomId", Value: room.ID.String()}}

	AddRoomMember(ctx)

	if recorder.Code != 403 {
		t.Fatalf("want 403 for cross-unit member add, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationAddRoomMemberAllowsUnitAdmin(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	candidate := testutil.MakeUser(t, "citizen")
	setUserUnit(t, candidate, unit.ID)

	setUserUnit(t, head, unit.ID)
	ctx, recorder := commJSONContext(t, "POST", "/communication/rooms/"+room.ID.String()+"/members", map[string]any{
		"userId": candidate.ID.String(),
	}, head)
	ctx.Params = gin.Params{{Key: "roomId", Value: room.ID.String()}}

	AddRoomMember(ctx)

	if recorder.Code != 201 {
		t.Fatalf("want 201 for unit admin adding member, got %d %s", recorder.Code, recorder.Body.String())
	}

	var member models.CommunicationRoomMember
	if err := config.DB.Where("room_id = ? AND user_id = ?", room.ID, candidate.ID).First(&member).Error; err != nil {
		t.Fatalf("expected membership row: %v", err)
	}
	if member.Status != "active" {
		t.Fatalf("want active membership, got %q", member.Status)
	}
}

func TestCommunicationRemoveRoomMemberRevokesAccess(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	member := testutil.MakeUser(t, "citizen")
	addRoomMember(t, room.ID, member.ID, "member", "active")

	setUserUnit(t, head, unit.ID)
	ctx, recorder := commJSONContext(t, "DELETE", "/communication/rooms/"+room.ID.String()+"/members/"+member.ID.String(), nil, head)
	ctx.Params = gin.Params{
		{Key: "roomId", Value: room.ID.String()},
		{Key: "userId", Value: member.ID.String()},
	}

	RemoveRoomMember(ctx)

	if recorder.Code != 200 {
		t.Fatalf("want 200 for unit admin removing member, got %d %s", recorder.Code, recorder.Body.String())
	}

	// The removed member can no longer post to the room.
	ctx2, recorder2 := commJSONContext(t, "POST", "/communication/messages", map[string]any{
		"roomId": room.ID.String(), "message": "still here?",
	}, member)

	SendMessage(ctx2)

	if recorder2.Code != 403 {
		t.Fatalf("want 403 for removed member message, got %d %s", recorder2.Code, recorder2.Body.String())
	}
}

func TestCommunicationGetRoomMembersRejectsNonMember(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	outsider := testutil.MakeUser(t, "citizen")
	ctx, recorder := commJSONContext(t, "GET", "/communication/rooms/"+room.ID.String()+"/members", nil, outsider)
	ctx.Params = gin.Params{{Key: "roomId", Value: room.ID.String()}}

	GetRoomMembers(ctx)

	if recorder.Code != 403 {
		t.Fatalf("want 403 for non-member member list, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationSyncMessagesRejectsNonMember(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	outsider := testutil.MakeUser(t, "citizen")
	ctx, recorder := commJSONContext(t, "POST", "/communication/sync", map[string]any{
		"roomId": room.ID.String(),
	}, outsider)

	SyncMessages(ctx)

	if recorder.Code != 403 {
		t.Fatalf("want 403 for non-member sync, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationGetSyncStatusRejectsNonMember(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	outsider := testutil.MakeUser(t, "citizen")
	ctx, recorder := commJSONContext(t, "GET", "/communication/rooms/"+room.ID.String()+"/sync-status", nil, outsider)
	ctx.Params = gin.Params{{Key: "roomId", Value: room.ID.String()}}

	GetSyncStatus(ctx)

	if recorder.Code != 403 {
		t.Fatalf("want 403 for non-member sync status, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationRoomAccessAllowedRejectsUnknownRoom(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	user := testutil.MakeUser(t, "citizen")
	ctx, recorder := commJSONContext(t, "GET", "/", nil, user)

	if roomAccessAllowed(ctx, user, uuid.New()) {
		t.Fatal("want unknown room to be denied")
	}
	if recorder.Code != 404 {
		t.Fatalf("want 404 for unknown room, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCommunicationRoomAccessAllowedAllowsSuperAdmin(t *testing.T) {
	if config.DB == nil {
		t.Skip("integration DB unavailable")
	}
	head := testutil.MakeUser(t, "unit_admin")
	unit := testutil.MakeUnit(t, head)
	room := makeRoom(t, unit, head)

	superAdmin := testutil.MakeUser(t, "super_admin")
	superAdmin.IsSuperAdmin = true
	ctx, _ := commJSONContext(t, "GET", "/", nil, superAdmin)

	if !roomAccessAllowed(ctx, superAdmin, room.ID) {
		t.Fatal("want super admin to be allowed")
	}
}
