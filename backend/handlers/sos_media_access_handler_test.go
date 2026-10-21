package handlers

import (
    "net/http"
    "testing"

    "github.com/gin-gonic/gin"

    "security-solution/config"
    "security-solution/internal/testutil"
    "security-solution/models"
)

func TestServeSOSAttachment_ScopedPrivateVideo(t *testing.T) {
    skipWithoutDB(t)
    testutil.TruncateAll(t)
    defer testutil.TruncateAll(t)
    storage := newMediaStore(t)
    owner := testutil.MakeUser(t, "citizen")
    outsider := testutil.MakeUser(t, "citizen")
    key, _ := storage.put("sos_media", "clip.mp4", []byte("test video data"))

    alert := models.SOSAlert{
        UserID: owner.ID, Status: "pending", Priority: "high", Severity: "normal",
        DispatchState: "open", Latitude: 9.0765, Longitude: 7.3986,
    }
    if err := config.DB.Create(&alert).Error; err != nil { t.Fatal(err) }
    link := models.MediaLink{ParentID: alert.ID, ParentType: MediaParentSOS, ObjectKey: key}
    if err := config.DB.Create(&link).Error; err != nil { t.Fatal(err) }

    params := gin.Params{{Key: "id", Value: alert.ID.String()}, {Key: "mediaId", Value: link.ID.String()}}
    target := "/sos/" + alert.ID.String() + "/media/" + link.ID.String() + "/file"

    ctx, rejected := mediaContext(t, "GET", target, params, outsider, nil)
    ServeSOSAttachment(ctx)
    if rejected.Code != http.StatusNotFound {
        t.Fatalf("unrelated user must not read SOS media: %d %s", rejected.Code, rejected.Body.String())
    }

    authorized, ok := mediaContext(t, "GET", target, params, owner, nil)
    ServeSOSAttachment(authorized)
    if ok.Code != http.StatusOK || ok.Body.String() != "test video data" {
        t.Fatalf("SOS owner must receive private media: %d %s", ok.Code, ok.Body.String())
    }
    if ok.Header().Get("Cache-Control") != "private, no-store" {
        t.Fatal("private SOS recording must not be cached publicly")
    }
}
