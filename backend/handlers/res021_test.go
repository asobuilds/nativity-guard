//go:build integration

package handlers

import (
	"bytes"
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

// replyContext mirrors reportContext: a POST body carrying the post id and
// reply content, with the authenticated user placed in the gin context so
// CreateForumReply can bind and authorize as in production.
func replyContext(postID uuid.UUID, user *models.User, content string) (*gin.Context, *httptest.ResponseRecorder) {
	payload, _ := json.Marshal(map[string]any{"postId": postID.String(), "content": content})
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/community/posts/"+postID.String()+"/reply", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("user", user)
	return ctx, recorder
}

// TestCreateForumReply_RejectsUnpublishedPost pins the status filter on the
// reply target lookup: a draft post is invisible to the reply path and must
// 404, leaving reply_count untouched at zero.
func TestCreateForumReply_RejectsUnpublishedPost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	author := testutil.MakeUser(t, "citizen")

	post := &models.ForumPost{
		Title:    "Draft post",
		Content:  "content",
		AuthorID: author.ID,
		Status:   "draft",
		GeoPlace: models.GeoPlace{Scope: ScopeWorld},
	}
	if err := config.DB.Create(post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	replier := testutil.MakeUser(t, "citizen")
	ctx, recorder := replyContext(post.ID, replier, "hello")
	CreateForumReply(ctx)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("want 404 for draft post, got %d %s", recorder.Code, recorder.Body.String())
	}

	var reloaded models.ForumPost
	if err := config.DB.First(&reloaded, "id = ?", post.ID).Error; err != nil {
		t.Fatalf("reload post: %v", err)
	}
	if reloaded.ReplyCount != 0 {
		t.Errorf("want reply_count 0 after rejected reply, got %d", reloaded.ReplyCount)
	}

	var replyCount int64
	if err := config.DB.Model(&models.ForumReply{}).
		Where("post_id = ?", post.ID).Count(&replyCount).Error; err != nil {
		t.Fatalf("count replies: %v", err)
	}
	if replyCount != 0 {
		t.Errorf("want 0 reply rows, got %d", replyCount)
	}
}

// TestCreateForumReply_AtomicReplyCount spins 10 concurrent callers at one
// published post and confirms the denormalised counter ends at exactly 10 — the
// atomic UPDATE (gorm.Expr("reply_count + 1")) must not lose races.
func TestCreateForumReply_AtomicReplyCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	author := testutil.MakeUser(t, "citizen")

	post := &models.ForumPost{
		Title:    "Published post",
		Content:  "content",
		AuthorID: author.ID,
		Status:   "published",
		GeoPlace: models.GeoPlace{Scope: ScopeWorld},
	}
	if err := config.DB.Create(post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	replier := testutil.MakeUser(t, "citizen")

	const n = 10
	var wg sync.WaitGroup
	var mu sync.Mutex
	var codes []int
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			ctx, recorder := replyContext(post.ID, replier, "reply")
			CreateForumReply(ctx)
			mu.Lock()
			codes = append(codes, recorder.Code)
			mu.Unlock()
		}()
	}
	wg.Wait()

	for _, code := range codes {
		if code != http.StatusCreated {
			t.Errorf("concurrent reply: want 201, got %d", code)
		}
	}

	var reloaded models.ForumPost
	if err := config.DB.First(&reloaded, "id = ?", post.ID).Error; err != nil {
		t.Fatalf("reload post: %v", err)
	}
	if reloaded.ReplyCount != n {
		t.Errorf("want reply_count %d after %d concurrent replies, got %d", n, n, reloaded.ReplyCount)
	}

	var replyRows int64
	if err := config.DB.Model(&models.ForumReply{}).
		Where("post_id = ?", post.ID).Count(&replyRows).Error; err != nil {
		t.Fatalf("count replies: %v", err)
	}
	if replyRows != n {
		t.Errorf("want %d reply rows, got %d", n, replyRows)
	}
}

// TestGetCommunityEvents_GeoScoped pins the LGA-scoped event feed: a citizen
// whose resolved place is Lagos / Ikeja (via a verified unit membership) must
// see only the Lagos event, never the Benue event.
func TestGetCommunityEvents_GeoScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	// A verified Lagos unit gives the citizen a resolved place of
	// {State:"Lagos", LGA:"Ikeja"} without touching User.UnitID, so the
	// citizen role filter keeps applying "unit_id IS NULL".
	lagosUnit := &models.SecurityUnit{
		Name:               "Lagos Unit " + uuid.NewString()[:8],
		Type:               "community",
		Status:             "active",
		IsVerified:         true,
		State:              "Lagos",
		LGA:                "Ikeja",
		RegistrationNumber: "REG-LAGOS-" + uuid.NewString()[:8],
	}
	if err := config.DB.Create(lagosUnit).Error; err != nil {
		t.Fatalf("create lagos unit: %v", err)
	}

	user := testutil.MakeUser(t, "citizen")
	testutil.MakeMembership(t, user, lagosUnit, models.UnitRoleOfficer, false)

	author := testutil.MakeUser(t, "citizen")
	now := time.Now().UTC()

	lagosEvent := &models.CommunityEvent{
		Title:       "Lagos Community Meeting",
		Description: "A meeting in Lagos",
		Location:    "Ikeja",
		EventDate:   now.Add(24 * time.Hour),
		Status:      "upcoming",
		AuthorID:    author.ID,
		GeoPlace:    models.GeoPlace{Scope: ScopeLGA, State: "Lagos", LGA: "Ikeja"},
	}
	if err := config.DB.Create(lagosEvent).Error; err != nil {
		t.Fatalf("create lagos event: %v", err)
	}

	benueEvent := &models.CommunityEvent{
		Title:       "Benue Community Meeting",
		Description: "A meeting in Benue",
		Location:    "Makurdi",
		EventDate:   now.Add(48 * time.Hour),
		Status:      "upcoming",
		AuthorID:    author.ID,
		GeoPlace:    models.GeoPlace{Scope: ScopeLGA, State: "Benue", LGA: "Makurdi"},
	}
	if err := config.DB.Create(benueEvent).Error; err != nil {
		t.Fatalf("create benue event: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/community/events", nil)
	ctx.Set("user", user)
	GetCommunityEvents(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d %s", recorder.Code, recorder.Body.String())
	}

	var resp struct {
		Events []struct {
			ID    uuid.UUID `json:"id"`
			Title string    `json:"title"`
		} `json:"events"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v %s", err, recorder.Body.String())
	}

	if len(resp.Events) != 1 {
		t.Fatalf("want 1 event for Lagos user, got %d: %+v", len(resp.Events), resp.Events)
	}
	if resp.Events[0].ID != lagosEvent.ID {
		t.Errorf("want lagos event %s, got %s", lagosEvent.ID, resp.Events[0].ID)
	}
	if resp.Events[0].Title != lagosEvent.Title {
		t.Errorf("want %q, got %q", lagosEvent.Title, resp.Events[0].Title)
	}
	for _, e := range resp.Events {
		if e.ID == benueEvent.ID {
			t.Error("Benue event must not be visible to a Lagos user")
		}
	}
}
