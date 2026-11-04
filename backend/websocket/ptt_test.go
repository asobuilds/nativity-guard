package websocket

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"security-solution/models"
)

func testClient(id, room, role string) *Client {
	return &Client{
		ID:       id,
		Room:     room,
		UserID:   uuid.NewString(),
		Username: id,
		Role:     role,
		Send:     make(chan Outbound, 8),
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}

func TestRequestFloorGrantsOneSpeakerPerRoom(t *testing.T) {
	m := NewPTTManager(nil, time.Minute)
	first := testClient("first", "unit-1", models.RoleOfficer)
	second := testClient("second", "unit-1", models.RoleOfficer)

	result := m.RequestFloor(first)
	if !result.Granted {
		t.Fatalf("first request should be granted, got %+v", result)
	}
	state := m.Snapshot("unit-1")
	if state.Status != "busy" || state.SpeakerID != first.ID {
		t.Fatalf("unexpected state after grant: %+v", state)
	}

	result = m.RequestFloor(second)
	if !result.Busy {
		t.Fatalf("second request should be busy, got %+v", result)
	}
	if result.State.SpeakerID != first.ID {
		t.Fatalf("busy state should name the first speaker, got %+v", result.State)
	}
	if !m.IsSpeaker(first) {
		t.Fatal("first client should be the current speaker")
	}
	if m.IsSpeaker(second) {
		t.Fatal("second client must not be the speaker")
	}
}

func TestFloorIsPerUnit(t *testing.T) {
	m := NewPTTManager(nil, time.Minute)
	one := testClient("one", "unit-1", models.RoleOfficer)
	two := testClient("two", "unit-2", models.RoleOfficer)

	if result := m.RequestFloor(one); !result.Granted {
		t.Fatalf("unit-1 should grant: %+v", result)
	}
	if result := m.RequestFloor(two); !result.Granted {
		t.Fatalf("unit-2 must not be blocked by unit-1: %+v", result)
	}
}

func TestReleaseRules(t *testing.T) {
	m := NewPTTManager(nil, time.Minute)
	speaker := testClient("speaker", "unit-1", models.RoleOfficer)
	other := testClient("other", "unit-1", models.RoleOfficer)

	if result := m.RequestFloor(speaker); !result.Granted {
		t.Fatalf("grant failed: %+v", result)
	}

	// A non-speaker cannot release someone else's floor.
	m.ReleaseFloor(other, "released")
	if state := m.Snapshot("unit-1"); state.Status != "busy" || state.SpeakerID != speaker.ID {
		t.Fatalf("non-speaker release must be ignored, got %+v", state)
	}

	// The speaker can release, and the next request is granted.
	m.ReleaseFloor(speaker, "released")
	state := m.Snapshot("unit-1")
	if state.Status != "idle" || state.SpeakerID != "" {
		t.Fatalf("expected idle after release, got %+v", state)
	}
	if result := m.RequestFloor(other); !result.Granted {
		t.Fatalf("floor should be available after release: %+v", result)
	}
}

func TestTimeoutReleasesFloor(t *testing.T) {
	m := NewPTTManager(nil, 25*time.Millisecond)
	speaker := testClient("speaker", "unit-1", models.RoleOfficer)
	next := testClient("next", "unit-1", models.RoleOfficer)

	if result := m.RequestFloor(speaker); !result.Granted {
		t.Fatalf("grant failed: %+v", result)
	}

	waitFor(t, time.Second, func() bool {
		return m.Snapshot("unit-1").Status == "idle"
	})
	if m.IsSpeaker(speaker) {
		t.Fatal("speaker should lose the floor after the timeout")
	}
	if result := m.RequestFloor(next); !result.Granted {
		t.Fatalf("floor should be available after timeout: %+v", result)
	}
}

func TestStaleTimeoutDoesNotReleaseNewSpeaker(t *testing.T) {
	m := NewPTTManager(nil, time.Minute)
	first := testClient("first", "unit-1", models.RoleOfficer)
	second := testClient("second", "unit-1", models.RoleOfficer)

	m.RequestFloor(first)
	m.ReleaseFloor(first, "released")
	m.RequestFloor(second)

	// A timer callback from the first grant must not clear the second speaker.
	m.Timeout("unit-1", 1)
	if state := m.Snapshot("unit-1"); state.SpeakerID != second.ID {
		t.Fatalf("stale timeout cleared the new speaker: %+v", state)
	}

	// The current grant's timeout does release it.
	current := m.Snapshot("unit-1")
	if current.RemainingSeconds == 0 {
		t.Fatal("expected a live grant")
	}
	// GrantID is not exposed in the snapshot; use a fresh manager grant to get
	// the current id via the room internals.
	room := m.room("unit-1")
	room.Mutex.Lock()
	grantID := room.GrantID
	room.Mutex.Unlock()
	m.Timeout("unit-1", grantID)
	if state := m.Snapshot("unit-1"); state.Status != "idle" {
		t.Fatalf("current timeout should release the floor, got %+v", state)
	}
}

func TestDisconnectReleasesFloor(t *testing.T) {
	m := NewPTTManager(nil, time.Minute)
	speaker := testClient("speaker", "unit-1", models.RoleOfficer)
	next := testClient("next", "unit-1", models.RoleOfficer)

	m.RequestFloor(speaker)
	m.Leave(speaker)
	if state := m.Snapshot("unit-1"); state.Status != "idle" {
		t.Fatalf("disconnect should release the floor, got %+v", state)
	}
	if result := m.RequestFloor(next); !result.Granted {
		t.Fatalf("next speaker should be granted after disconnect: %+v", result)
	}
}

func TestSuperAdminRequiresCommandOverrideToTransmit(t *testing.T) {
	m := NewPTTManager(nil, time.Minute)
	superAdmin := testClient("super", "unit-1", models.RoleSuperAdmin)

	result := m.RequestFloor(superAdmin)
	if !result.Denied || result.Reason != "command_override_required" {
		t.Fatalf("super admin without override must be denied, got %+v", result)
	}
	if m.Snapshot("unit-1").Status != "idle" {
		t.Fatal("denied request must not take the floor")
	}

	superAdmin.CommandOverride.Store(true)
	if result := m.RequestFloor(superAdmin); !result.Granted {
		t.Fatalf("super admin with override should be granted, got %+v", result)
	}
}

func TestSnapshotCountsDown(t *testing.T) {
	m := NewPTTManager(nil, time.Minute)
	speaker := testClient("speaker", "unit-1", models.RoleOfficer)
	m.RequestFloor(speaker)

	state := m.Snapshot("unit-1")
	if state.RemainingSeconds != 60 {
		t.Fatalf("expected 60 seconds remaining, got %d", state.RemainingSeconds)
	}
	if state.ExpiresAt == nil {
		t.Fatal("expected an expiry on a live grant")
	}
}
