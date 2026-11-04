//go:build integration

package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

type taskForceFixture struct {
	hostHead     *models.User
	hostUnit     *models.SecurityUnit
	memberHead   *models.User
	memberUnit   *models.SecurityUnit
	outsiderHead *models.User
	outsiderUnit *models.SecurityUnit
	taskForce    *models.TaskForce
}

func taskForceContext(
	t *testing.T,
	method, target string,
	body map[string]any,
	user *models.User,
	params gin.Params,
) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	if user != nil {
		ctx.Set("user", user)
	}
	if params != nil {
		ctx.Params = params
	}
	return ctx, recorder
}

func taskForceIDParams(id uuid.UUID) gin.Params {
	return gin.Params{{Key: "id", Value: id.String()}}
}

func decodeTaskForceResponse(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
}

func newTaskForceFixture(t *testing.T, name string) *taskForceFixture {
	t.Helper()

	hostHead := testutil.MakeUser(t, models.RoleUnitAdmin)
	hostUnit := testutil.MakeUnit(t, hostHead)
	setUserUnit(t, hostHead, hostUnit.ID)

	memberHead := testutil.MakeUser(t, models.RoleUnitAdmin)
	memberUnit := testutil.MakeUnit(t, memberHead)
	setUserUnit(t, memberHead, memberUnit.ID)

	outsiderHead := testutil.MakeUser(t, models.RoleUnitAdmin)
	outsiderUnit := testutil.MakeUnit(t, outsiderHead)
	setUserUnit(t, outsiderHead, outsiderUnit.ID)

	taskForce := makeTaskForceRow(t, hostHead, hostUnit, name)
	return &taskForceFixture{
		hostHead:     hostHead,
		hostUnit:     hostUnit,
		memberHead:   memberHead,
		memberUnit:   memberUnit,
		outsiderHead: outsiderHead,
		outsiderUnit: outsiderUnit,
		taskForce:    taskForce,
	}
}

func makeTaskForceRow(t *testing.T, creator *models.User, hostUnit *models.SecurityUnit, name string) *models.TaskForce {
	t.Helper()
	taskForce := &models.TaskForce{
		Name:       name + " " + uuid.NewString()[:8],
		HostUnitID: hostUnit.ID,
		Status:     models.TaskForceStatusActive,
		CreatedBy:  creator.ID,
	}
	if err := config.DB.Create(taskForce).Error; err != nil {
		t.Fatalf("create task force: %v", err)
	}
	now := time.Now().UTC()
	member := &models.TaskForceMember{
		TaskForceID: taskForce.ID,
		UnitID:      hostUnit.ID,
		Role:        models.TaskForceMemberRoleHost,
		Status:      models.TaskForceMemberStatusActive,
		AcceptedAt:  &now,
	}
	if err := config.DB.Create(member).Error; err != nil {
		t.Fatalf("create host member: %v", err)
	}
	return taskForce
}

func makeTaskForceOfficer(t *testing.T, unit *models.SecurityUnit, status string) *models.Officer {
	t.Helper()
	officer := &models.Officer{
		UnitID:        unit.ID,
		Name:          "Officer " + uuid.NewString()[:8],
		Rank:          "Constable",
		BadgeNumber:   "BADGE-" + uuid.NewString()[:8],
		UnitOfficerID: "UO-" + uuid.NewString()[:8],
		Role:          "patrol",
		Status:        status,
		JoinedDate:    time.Now().UTC(),
	}
	if err := config.DB.Create(officer).Error; err != nil {
		t.Fatalf("create officer: %v", err)
	}
	return officer
}

func inviteTaskForceUnit(t *testing.T, taskForceID uuid.UUID, invitee *models.User, unitID uuid.UUID) string {
	t.Helper()
	ctx, recorder := taskForceContext(t, http.MethodPost, "/taskforce/"+taskForceID.String()+"/invite",
		map[string]any{"unitId": unitID.String()}, invitee, taskForceIDParams(taskForceID))
	InviteTaskForceUnit(ctx)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("invite failed: %d %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Code string `json:"code"`
	}
	decodeTaskForceResponse(t, recorder, &payload)
	if payload.Code == "" {
		t.Fatal("invite response omitted code")
	}
	return payload.Code
}

func joinTaskForceWithCode(t *testing.T, taskForceID uuid.UUID, user *models.User, code string) *httptest.ResponseRecorder {
	t.Helper()
	ctx, recorder := taskForceContext(t, http.MethodPost, "/taskforce/"+taskForceID.String()+"/join",
		map[string]any{"code": code}, user, taskForceIDParams(taskForceID))
	JoinTaskForce(ctx)
	return recorder
}

func enlistTaskForceOfficers(t *testing.T, taskForceID uuid.UUID, host *models.User, officerIDs ...uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	raw := make([]string, 0, len(officerIDs))
	for _, id := range officerIDs {
		raw = append(raw, id.String())
	}
	ctx, recorder := taskForceContext(t, http.MethodPost, "/taskforce/"+taskForceID.String()+"/enlist",
		map[string]any{"officerIds": raw}, host, taskForceIDParams(taskForceID))
	EnlistTaskForceOfficers(ctx)
	return recorder
}

func TestTaskForceCreateRequiresStrictHeadAdmin(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	hostHead := testutil.MakeUser(t, models.RoleUnitAdmin)
	hostUnit := testutil.MakeUnit(t, hostHead)
	setUserUnit(t, hostHead, hostUnit.ID)

	ctx, recorder := taskForceContext(t, http.MethodPost, "/taskforce", map[string]any{
		"name":       "Joint Operation",
		"hostUnitId": hostUnit.ID.String(),
	}, hostHead, nil)
	CreateTaskForce(ctx)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("head admin create: want 201, got %d %s", recorder.Code, recorder.Body.String())
	}

	plainAdmin := testutil.MakeUser(t, models.RoleUnitAdmin)
	setUserUnit(t, plainAdmin, hostUnit.ID)
	testutil.MakeMembership(t, plainAdmin, hostUnit, models.UnitRoleAdmin, false)

	ctx, recorder = taskForceContext(t, http.MethodPost, "/taskforce", map[string]any{
		"name":       "Sneaky Operation",
		"hostUnitId": hostUnit.ID.String(),
	}, plainAdmin, nil)
	CreateTaskForce(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("plain unit admin create: want 403, got %d %s", recorder.Code, recorder.Body.String())
	}

	otherHead := testutil.MakeUser(t, models.RoleUnitAdmin)
	otherUnit := testutil.MakeUnit(t, otherHead)
	setUserUnit(t, otherHead, otherUnit.ID)

	ctx, recorder = taskForceContext(t, http.MethodPost, "/taskforce", map[string]any{
		"name":       "Cross Unit Operation",
		"hostUnitId": hostUnit.ID.String(),
	}, otherHead, nil)
	CreateTaskForce(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("foreign head admin create: want 403, got %d %s", recorder.Code, recorder.Body.String())
	}

	superAdmin := testutil.MakeUser(t, models.RoleSuperAdmin)
	ctx, recorder = taskForceContext(t, http.MethodPost, "/taskforce", map[string]any{
		"name":       "Supervised Operation",
		"hostUnitId": hostUnit.ID.String(),
	}, superAdmin, nil)
	CreateTaskForce(ctx)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("super admin create: want 201, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestTaskForceInviteStoresHashAndJoinActivatesMember(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	fixture := newTaskForceFixture(t, "Joint Patrol")

	code := inviteTaskForceUnit(t, fixture.taskForce.ID, fixture.hostHead, fixture.memberUnit.ID)

	var invite models.TaskForceInvite
	if err := config.DB.Where("task_force_id = ? AND unit_id = ?", fixture.taskForce.ID, fixture.memberUnit.ID).
		First(&invite).Error; err != nil {
		t.Fatalf("load invite: %v", err)
	}
	expectedHash := sha256.Sum256([]byte(code))
	if invite.CodeHash != hex.EncodeToString(expectedHash[:]) {
		t.Fatalf("invite did not store sha256 hash")
	}
	if invite.CodeHash == code {
		t.Fatal("plaintext invite code was stored")
	}
	if invite.Status != models.TaskForceInviteStatusPending {
		t.Fatalf("invite status = %q, want pending", invite.Status)
	}

	var member models.TaskForceMember
	if err := config.DB.Where("task_force_id = ? AND unit_id = ?", fixture.taskForce.ID, fixture.memberUnit.ID).
		First(&member).Error; err != nil {
		t.Fatalf("load invited member: %v", err)
	}
	if member.Status != models.TaskForceMemberStatusInvited {
		t.Fatalf("member status = %q, want invited", member.Status)
	}

	recorder := joinTaskForceWithCode(t, fixture.taskForce.ID, fixture.memberHead, code)
	if recorder.Code != http.StatusOK {
		t.Fatalf("join: want 200, got %d %s", recorder.Code, recorder.Body.String())
	}

	if err := config.DB.First(&member, "id = ?", member.ID).Error; err != nil {
		t.Fatalf("reload member: %v", err)
	}
	if member.Status != models.TaskForceMemberStatusActive || member.AcceptedAt == nil {
		t.Fatalf("member not activated: status=%q acceptedAt=%v", member.Status, member.AcceptedAt)
	}

	if err := config.DB.First(&invite, "id = ?", invite.ID).Error; err != nil {
		t.Fatalf("reload invite: %v", err)
	}
	if invite.Status != models.TaskForceInviteStatusAccepted || invite.UseCount != 1 {
		t.Fatalf("invite not consumed: status=%q uses=%d", invite.Status, invite.UseCount)
	}
}

func TestTaskForceInviteAndJoinAuthorization(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	fixture := newTaskForceFixture(t, "Authorized Joint Patrol")

	ctx, recorder := taskForceContext(t, http.MethodPost, "/taskforce/"+fixture.taskForce.ID.String()+"/invite",
		map[string]any{"unitId": fixture.memberUnit.ID.String()}, fixture.memberHead, taskForceIDParams(fixture.taskForce.ID))
	InviteTaskForceUnit(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("member unit head invite: want 403, got %d %s", recorder.Code, recorder.Body.String())
	}

	ctx, recorder = taskForceContext(t, http.MethodPost, "/taskforce/"+fixture.taskForce.ID.String()+"/invite",
		map[string]any{"unitId": fixture.memberUnit.ID.String()}, fixture.outsiderHead, taskForceIDParams(fixture.taskForce.ID))
	InviteTaskForceUnit(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("outsider head invite: want 403, got %d %s", recorder.Code, recorder.Body.String())
	}

	code := inviteTaskForceUnit(t, fixture.taskForce.ID, fixture.hostHead, fixture.memberUnit.ID)

	ctx, recorder = taskForceContext(t, http.MethodGet, "/taskforce/"+fixture.taskForce.ID.String(), nil, fixture.memberHead, taskForceIDParams(fixture.taskForce.ID))
	GetTaskForce(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("invited head view: want 200, got %d %s", recorder.Code, recorder.Body.String())
	}

	ctx, recorder = taskForceContext(t, http.MethodGet, "/taskforce/"+fixture.taskForce.ID.String(), nil, fixture.outsiderHead, taskForceIDParams(fixture.taskForce.ID))
	GetTaskForce(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("outsider head view: want 403, got %d %s", recorder.Code, recorder.Body.String())
	}

	recorder = joinTaskForceWithCode(t, fixture.taskForce.ID, fixture.outsiderHead, code)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("wrong unit join: want 403, got %d %s", recorder.Code, recorder.Body.String())
	}

	targetOfficer := testutil.MakeUser(t, models.RoleOfficer)
	testutil.MakeMembership(t, targetOfficer, fixture.memberUnit, models.UnitRoleOfficer, false)
	setUserUnit(t, targetOfficer, fixture.memberUnit.ID)

	recorder = joinTaskForceWithCode(t, fixture.taskForce.ID, targetOfficer, code)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("non-head join: want 403, got %d %s", recorder.Code, recorder.Body.String())
	}

	recorder = joinTaskForceWithCode(t, fixture.taskForce.ID, fixture.memberHead, code)
	if recorder.Code != http.StatusOK {
		t.Fatalf("target head join: want 200, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestTaskForceEnlistAuthorizationAndPartialResults(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	fixture := newTaskForceFixture(t, "Enlistment Operation")
	code := inviteTaskForceUnit(t, fixture.taskForce.ID, fixture.hostHead, fixture.memberUnit.ID)
	recorder := joinTaskForceWithCode(t, fixture.taskForce.ID, fixture.memberHead, code)
	if recorder.Code != http.StatusOK {
		t.Fatalf("join: %d %s", recorder.Code, recorder.Body.String())
	}

	memberOfficer := makeTaskForceOfficer(t, fixture.memberUnit, "active")
	inactiveOfficer := makeTaskForceOfficer(t, fixture.memberUnit, "inactive")
	outsiderOfficer := makeTaskForceOfficer(t, fixture.outsiderUnit, "active")

	recorder = enlistTaskForceOfficers(t, fixture.taskForce.ID, fixture.memberHead, memberOfficer.ID)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("member head enlist: want 403, got %d %s", recorder.Code, recorder.Body.String())
	}

	ctx, recorder := taskForceContext(t, http.MethodPost, "/taskforce/"+fixture.taskForce.ID.String()+"/enlist",
		map[string]any{"officerIds": []string{
			memberOfficer.ID.String(),
			uuid.NewString(),
			outsiderOfficer.ID.String(),
			inactiveOfficer.ID.String(),
			memberOfficer.ID.String(),
		}}, fixture.hostHead, taskForceIDParams(fixture.taskForce.ID))
	EnlistTaskForceOfficers(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("host enlist: want 200, got %d %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Enlisted int                     `json:"enlisted"`
		Results  []taskForceEnlistResult `json:"results"`
	}
	decodeTaskForceResponse(t, recorder, &payload)
	if payload.Enlisted != 1 {
		t.Fatalf("enlisted = %d, want 1. results=%+v", payload.Enlisted, payload.Results)
	}

	var count int64
	config.DB.Model(&models.TaskForceOfficer{}).
		Where("task_force_id = ? AND status = ?", fixture.taskForce.ID, models.TaskForceOfficerStatusActive).
		Count(&count)
	if count != 1 {
		t.Fatalf("active enlisted rows = %d, want 1", count)
	}

	recorder = enlistTaskForceOfficers(t, fixture.taskForce.ID, fixture.hostHead, memberOfficer.ID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("repeat enlist: want 200, got %d %s", recorder.Code, recorder.Body.String())
	}
	decodeTaskForceResponse(t, recorder, &payload)
	if payload.Enlisted != 0 || len(payload.Results) != 1 || payload.Results[0].Status != "already_active" {
		t.Fatalf("repeat enlist result = %+v, want already_active", payload.Results)
	}
}

func TestTaskForceOneOfficerOneActiveTaskForce(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	fixture := newTaskForceFixture(t, "First Operation")
	code := inviteTaskForceUnit(t, fixture.taskForce.ID, fixture.hostHead, fixture.memberUnit.ID)
	if recorder := joinTaskForceWithCode(t, fixture.taskForce.ID, fixture.memberHead, code); recorder.Code != http.StatusOK {
		t.Fatalf("join: %d %s", recorder.Code, recorder.Body.String())
	}

	officer := makeTaskForceOfficer(t, fixture.memberUnit, "active")
	recorder := enlistTaskForceOfficers(t, fixture.taskForce.ID, fixture.hostHead, officer.ID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("first enlist: %d %s", recorder.Code, recorder.Body.String())
	}

	second := makeTaskForceRow(t, fixture.hostHead, fixture.hostUnit, "Second Operation")
	now := time.Now().UTC()
	if err := config.DB.Create(&models.TaskForceMember{
		TaskForceID: second.ID,
		UnitID:      fixture.memberUnit.ID,
		Role:        models.TaskForceMemberRoleMember,
		Status:      models.TaskForceMemberStatusActive,
		AcceptedAt:  &now,
	}).Error; err != nil {
		t.Fatalf("add second member unit: %v", err)
	}

	recorder = enlistTaskForceOfficers(t, second.ID, fixture.hostHead, officer.ID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("second enlist: %d %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Enlisted int                     `json:"enlisted"`
		Results  []taskForceEnlistResult `json:"results"`
	}
	decodeTaskForceResponse(t, recorder, &payload)
	if payload.Enlisted != 0 || len(payload.Results) != 1 || payload.Results[0].Status != "conflict" {
		t.Fatalf("second enlist result = %+v, want conflict", payload.Results)
	}

	var activeCount int64
	config.DB.Model(&models.TaskForceOfficer{}).
		Where("officer_id = ? AND status = ?", officer.ID, models.TaskForceOfficerStatusActive).
		Count(&activeCount)
	if activeCount != 1 {
		t.Fatalf("active task-force assignments = %d, want 1", activeCount)
	}
}

func TestTaskForceConcurrentJoinUsesInviteOnce(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	fixture := newTaskForceFixture(t, "Concurrent Join")
	code := inviteTaskForceUnit(t, fixture.taskForce.ID, fixture.hostHead, fixture.memberUnit.ID)

	const callers = 12
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		ok       int
		rejected int
		other    int
		start    = make(chan struct{})
	)

	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			recorder := joinTaskForceWithCode(t, fixture.taskForce.ID, fixture.memberHead, code)
			mu.Lock()
			switch recorder.Code {
			case http.StatusOK:
				ok++
			case http.StatusGone, http.StatusConflict, http.StatusNotFound:
				rejected++
			default:
				other++
			}
			mu.Unlock()
		}()
	}

	close(start)
	wg.Wait()

	if other != 0 {
		t.Fatalf("unexpected join responses: %d", other)
	}
	if ok != 1 {
		t.Fatalf("successful joins = %d, want 1 (rejected=%d)", ok, rejected)
	}

	var member models.TaskForceMember
	if err := config.DB.Where("task_force_id = ? AND unit_id = ?", fixture.taskForce.ID, fixture.memberUnit.ID).
		First(&member).Error; err != nil {
		t.Fatalf("load member: %v", err)
	}
	if member.Status != models.TaskForceMemberStatusActive {
		t.Fatalf("member status = %q, want active", member.Status)
	}
}

func TestTaskForceChannelAndWalkieAuthorization(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	fixture := newTaskForceFixture(t, "Channel Operation")
	code := inviteTaskForceUnit(t, fixture.taskForce.ID, fixture.hostHead, fixture.memberUnit.ID)
	if recorder := joinTaskForceWithCode(t, fixture.taskForce.ID, fixture.memberHead, code); recorder.Code != http.StatusOK {
		t.Fatalf("join: %d %s", recorder.Code, recorder.Body.String())
	}

	memberOfficer := testutil.MakeUser(t, models.RoleOfficer)
	testutil.MakeMembership(t, memberOfficer, fixture.memberUnit, models.UnitRoleOfficer, false)
	setUserUnit(t, memberOfficer, fixture.memberUnit.ID)

	ctx, recorder := taskForceContext(t, http.MethodGet, "/taskforce/"+fixture.taskForce.ID.String(), nil,
		memberOfficer, taskForceIDParams(fixture.taskForce.ID))
	GetTaskForce(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("member get: want 200, got %d %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Channel     *models.CommunicationRoom `json:"channel"`
		CanTransmit bool                      `json:"canTransmit"`
	}
	decodeTaskForceResponse(t, recorder, &payload)
	if payload.Channel == nil || payload.Channel.Type != models.TaskForceRoomType {
		t.Fatalf("task force channel = %+v", payload.Channel)
	}
	if payload.Channel.UnitID != fixture.hostUnit.ID {
		t.Fatalf("channel unit = %s, want host unit %s", payload.Channel.UnitID, fixture.hostUnit.ID)
	}
	if !payload.CanTransmit {
		t.Fatal("member unit officer should be able to transmit")
	}

	outsiderOfficer := testutil.MakeUser(t, models.RoleOfficer)
	testutil.MakeMembership(t, outsiderOfficer, fixture.outsiderUnit, models.UnitRoleOfficer, false)
	setUserUnit(t, outsiderOfficer, fixture.outsiderUnit.ID)

	ctx, recorder = taskForceContext(t, http.MethodGet, "/taskforce/"+fixture.taskForce.ID.String(), nil,
		outsiderOfficer, taskForceIDParams(fixture.taskForce.ID))
	GetTaskForce(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("outsider get: want 403, got %d %s", recorder.Code, recorder.Body.String())
	}

	taskForce, err := loadTaskForce(fixture.taskForce.ID)
	if err != nil {
		t.Fatalf("load task force: %v", err)
	}
	if !canJoinTaskForceChannel(memberOfficer, taskForce) {
		t.Fatal("member unit officer should join the task-force channel")
	}
	if canJoinTaskForceChannel(outsiderOfficer, taskForce) {
		t.Fatal("outsider should not join the task-force channel")
	}
	if canJoinTaskForceChannel(testutil.MakeUser(t, models.RoleCitizen), taskForce) {
		t.Fatal("citizen should not join the task-force channel")
	}
}

func TestTaskForceLeaveAndDisbandReleaseOfficers(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	fixture := newTaskForceFixture(t, "Lifecycle Operation")
	if _, err := ensureTaskForceChannel(fixture.taskForce, fixture.hostHead.ID); err != nil {
		t.Fatalf("ensure channel: %v", err)
	}
	code := inviteTaskForceUnit(t, fixture.taskForce.ID, fixture.hostHead, fixture.memberUnit.ID)
	if recorder := joinTaskForceWithCode(t, fixture.taskForce.ID, fixture.memberHead, code); recorder.Code != http.StatusOK {
		t.Fatalf("join: %d %s", recorder.Code, recorder.Body.String())
	}

	officer := makeTaskForceOfficer(t, fixture.memberUnit, "active")
	if recorder := enlistTaskForceOfficers(t, fixture.taskForce.ID, fixture.hostHead, officer.ID); recorder.Code != http.StatusOK {
		t.Fatalf("enlist: %d %s", recorder.Code, recorder.Body.String())
	}

	ctx, recorder := taskForceContext(t, http.MethodPost, "/taskforce/"+fixture.taskForce.ID.String()+"/leave",
		nil, fixture.hostHead, taskForceIDParams(fixture.taskForce.ID))
	LeaveTaskForce(ctx)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("host leave: want 409, got %d %s", recorder.Code, recorder.Body.String())
	}

	ctx, recorder = taskForceContext(t, http.MethodPost, "/taskforce/"+fixture.taskForce.ID.String()+"/leave",
		nil, fixture.memberHead, taskForceIDParams(fixture.taskForce.ID))
	LeaveTaskForce(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("member leave: want 200, got %d %s", recorder.Code, recorder.Body.String())
	}

	var officerRow models.TaskForceOfficer
	if err := config.DB.Where("task_force_id = ? AND officer_id = ?", fixture.taskForce.ID, officer.ID).
		First(&officerRow).Error; err != nil {
		t.Fatalf("load officer row: %v", err)
	}
	if officerRow.Status != models.TaskForceOfficerStatusRemoved || officerRow.RemovedAt == nil {
		t.Fatalf("officer not released on leave: status=%q removedAt=%v", officerRow.Status, officerRow.RemovedAt)
	}

	officer2 := makeTaskForceOfficer(t, fixture.hostUnit, "active")
	if recorder := enlistTaskForceOfficers(t, fixture.taskForce.ID, fixture.hostHead, officer2.ID); recorder.Code != http.StatusOK {
		t.Fatalf("host enlist: %d %s", recorder.Code, recorder.Body.String())
	}

	ctx, recorder = taskForceContext(t, http.MethodPost, "/taskforce/"+fixture.taskForce.ID.String()+"/disband",
		nil, fixture.hostHead, taskForceIDParams(fixture.taskForce.ID))
	DisbandTaskForce(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("disband: want 200, got %d %s", recorder.Code, recorder.Body.String())
	}

	var taskForce models.TaskForce
	if err := config.DB.First(&taskForce, "id = ?", fixture.taskForce.ID).Error; err != nil {
		t.Fatalf("reload task force: %v", err)
	}
	if taskForce.Status != models.TaskForceStatusDisbanded {
		t.Fatalf("task force status = %q, want disbanded", taskForce.Status)
	}

	var activeRows int64
	config.DB.Model(&models.TaskForceOfficer{}).
		Where("task_force_id = ? AND status = ?", fixture.taskForce.ID, models.TaskForceOfficerStatusActive).
		Count(&activeRows)
	if activeRows != 0 {
		t.Fatalf("active officer rows after disband = %d, want 0", activeRows)
	}

	var room models.CommunicationRoom
	if err := config.DB.Model(&models.CommunicationRoom{}).
		Joins("JOIN task_force_channels ON task_force_channels.room_id = communication_rooms.id").
		Where("task_force_channels.task_force_id = ?", fixture.taskForce.ID).
		First(&room).Error; err != nil {
		t.Fatalf("load task force room: %v", err)
	}
	if room.Status != "closed" {
		t.Fatalf("room status after disband = %q, want closed", room.Status)
	}
}
