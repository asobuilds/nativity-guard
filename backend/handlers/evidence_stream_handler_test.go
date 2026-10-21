package handlers

import (
    "net/http"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "security-solution/config"
    "security-solution/internal/testutil"
    "security-solution/models"
)

func TestServeCaseEvidenceRequiresAuthentication(t *testing.T) {
    ctx, rec := mediaContext(t, "GET", "/evidence/case/123/file", gin.Params{
        {Key: "caseId", Value: uuid.NewString()},
        {Key: "evidenceId", Value: uuid.NewString()},
    }, nil, nil)
    ServeCaseEvidence(ctx)
    if rec.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401, got %d %s", rec.Code, rec.Body.String())
    }
}

func TestServeCaseEvidencePrivateAudioAccess(t *testing.T) {
    skipWithoutDB(t)
    testutil.TruncateAll(t)
    defer testutil.TruncateAll(t)
    store := newMediaStore(t)

    reporter := testutil.MakeUser(t, "citizen")
    stranger := testutil.MakeUser(t, "citizen")
    unit := testutil.MakeUnit(t, nil)
    caseRecord := testutil.MakeCase(t, reporter, unit)
    voiceBytes := []byte("sample audio bytes stored privately")
    key, _ := store.put("evidence", "voice.m4a", voiceBytes)
    item := models.Evidence{
        CaseID: caseRecord.ID,
        UploadedBy: reporter.ID,
        Type: "audio",
        FileURL: key,
        FilePath: key,
        MimeType: "audio/mp4",
    }
    if err := config.DB.Create(&item).Error; err != nil { t.Fatal(err) }
    path := "/evidence/case/" + caseRecord.ID.String() + "/" + item.ID.String() + "/file"
    params := gin.Params{
        {Key: "caseId", Value: caseRecord.ID.String()},
        {Key: "evidenceId", Value: item.ID.String()},
    }

    denied, rec := mediaContext(t, "GET", path, params, stranger, nil)
    ServeCaseEvidence(denied)
    if rec.Code != http.StatusForbidden {
        t.Fatalf("unauthorized case access must be refused, got %d %s", rec.Code, rec.Body.String())
    }

    authorized, okRec := mediaContext(t, "GET", path, params, reporter, nil)
    ServeCaseEvidence(authorized)
    if okRec.Code != http.StatusOK {
        t.Fatalf("reporter should access own evidence, got %d %s", okRec.Code, okRec.Body.String())
    }
    if okRec.Body.String() != string(voiceBytes) {
        t.Fatalf("stored audio bytes mismatch")
    }
    if okRec.Header().Get("Content-Type") != "audio/mp4" {
        t.Fatalf("wrong audio MIME: %q", okRec.Header().Get("Content-Type"))
    }
    if okRec.Header().Get("Cache-Control") != "private, no-store" {
        t.Fatal("private evidence must never be cached publicly")
    }
}
