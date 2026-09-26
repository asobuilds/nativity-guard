package routes

import (
	"github.com/gin-gonic/gin"
	"security-solution/handlers"
	"security-solution/middleware"
)

func SetupRoutes(router *gin.Engine) {
	// Health check endpoint — PUBLIC, no auth. External monitors
	// (Render, UptimeRobot) poll this to determine if the server is
	// alive and its subsystems are healthy.
	router.GET("/health", handlers.GetHealth)

	// API v1 routes
	api := router.Group("/api/v1")
	{
		// Public routes (no authentication required)
		api.GET("/public/cases", middleware.RateLimitGeneral(), handlers.GetPublicCases)
		api.GET("/public/units", middleware.RateLimitGeneral(), handlers.GetPublicUnits)
		api.GET("/public/units/:id/bank-accounts", handlers.GetPublicBankAccounts)
		api.GET("/public/units/:id/ledger", middleware.RateLimitGeneral(), handlers.GetPublicUnitLedger)
		api.GET("/public/units/:id/financial-years", handlers.GetPublicFinancialYears)
		api.GET("/public/units/:id/financial-summary", handlers.GetCurrentYearSummary)
		api.GET("/public/platform/donation-info", handlers.GetPlatformDonationInfo)
		api.POST("/public/platform/donations", handlers.CreatePlatformDonation)
		api.GET("/public/platform/supporters", handlers.ListPublicSupporters)
		api.GET("/public/leaderboard", handlers.GetPublicLeaderboard)
		api.GET("/public/units/suggest", handlers.SuggestUnits)
		api.GET("/public/officers/:id/rating", middleware.RateLimitGeneral(), handlers.GetOfficerRating)
		api.GET("/public/units/:id/rating", middleware.RateLimitGeneral(), handlers.GetUnitRating)
		api.GET("/public/blueprint", handlers.GetBlueprint)
		api.GET("/public/units/:id/auth", handlers.GetPublicUnitAuth)
		api.POST("/invites/validate", middleware.RateLimitAuth(), handlers.ValidateInvite)

		// Auth routes
		authHandler := handlers.NewAuthHandler()
		auth := api.Group("/auth")
		{
			auth.POST("/register", middleware.RateLimitAuth(), authHandler.Register)
			auth.POST("/login", middleware.RateLimitAuth(), authHandler.Login)
			auth.POST("/logout", middleware.RateLimitAuth(), middleware.AuthMiddleware(), authHandler.Logout)
			auth.GET("/profile", middleware.AuthMiddleware(), authHandler.GetProfile)
			auth.POST("/change-password", middleware.RateLimitAuth(), middleware.AuthMiddleware(), authHandler.ChangePassword)
			auth.POST("/refresh", middleware.RateLimitAuth(), authHandler.Refresh)
			auth.GET("/sessions", middleware.AuthMiddleware(), authHandler.ListSessions)
			auth.DELETE("/sessions/:jti", middleware.AuthMiddleware(), authHandler.RevokeSession)
			auth.DELETE("/sessions", middleware.AuthMiddleware(), authHandler.RevokeAllSessions)

			// Account lifecycle (Wave 5c)
			auth.POST("/forgot-password", middleware.RateLimitOTP(), authHandler.ForgotPassword)
			auth.POST("/reset-password", middleware.RateLimitOTP(), authHandler.ResetPassword)
			auth.DELETE("/account", middleware.AuthMiddleware(), authHandler.DeleteAccount)
			auth.POST("/account/cancel-deletion", middleware.AuthMiddleware(), authHandler.CancelDeletion)
		}

		// OTP routes
		otp := api.Group("/otp")
		{
			otp.POST("/send", middleware.RateLimitOTP(), handlers.SendOTP)
			otp.POST("/verify", middleware.RateLimitOTP(), handlers.VerifyOTP)
			otp.POST("/resend", middleware.RateLimitOTP(), handlers.ResendOTP)
		}

		// Unit routes
		units := api.Group("/units")
		{
			units.GET("/nearby", middleware.AuthMiddleware(), middleware.RateLimitMap(), handlers.GetNearbyUnits)
			units.GET("/by-location", middleware.AuthMiddleware(), middleware.RateLimitMap(), handlers.GetUnitsByLocation)
			units.GET("", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.GetAllUnits)

			units.POST("/apply", middleware.AuthMiddleware(), handlers.ApplyForSecurityUnit)
			units.GET("/my-memberships", middleware.AuthMiddleware(), handlers.GetMyUnitMembership)
			units.POST("/government-id", middleware.AuthMiddleware(), handlers.SubmitGovernmentID)

			units.GET("/:id", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.GetUnitByID)
			units.POST("", middleware.AuthMiddleware(), handlers.CreateUnit)
			units.PUT("/:id", middleware.AuthMiddleware(), handlers.UpdateUnit)
units.POST("/:id/elections", middleware.AuthMiddleware(), middleware.IdempotencyMiddleware(), handlers.OpenAdminElection)
    units.POST("/:id/head-admin-elections", middleware.AuthMiddleware(), middleware.IdempotencyMiddleware(), handlers.OpenHeadAdminElection)
    units.POST("/:id/revocations", middleware.AuthMiddleware(), middleware.IdempotencyMiddleware(), handlers.OpenRevocationCycle)
    units.GET("/:id/auth", middleware.AuthMiddleware(), handlers.GetUnitAuth)
    units.PUT("/:id/auth", middleware.AuthMiddleware(), handlers.UpsertUnitAuth)
    units.GET("/:id/officers", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.GetOfficersByUnit)
    units.GET("/:id/officers/ranking", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.GetOfficersInUnitRanking)
    units.GET("/:id/governance-audit", middleware.AuthMiddleware(), handlers.GetGovernanceAudit)
	}

	invites := api.Group("/invites")
	{
		invites.POST("", middleware.AuthMiddleware(), middleware.RateLimitInvite(), handlers.CreateInvite)
	}

	users := api.Group("/users")
	{
		users.POST("/me/avatar", middleware.AuthMiddleware(), middleware.UploadValidationMiddleware("image"), handlers.UploadAvatar)
		users.POST("/me/cover", middleware.AuthMiddleware(), middleware.UploadValidationMiddleware("image"), handlers.UploadCover)
		users.DELETE("/me/avatar", middleware.AuthMiddleware(), handlers.DeleteAvatar)
		users.DELETE("/me/cover", middleware.AuthMiddleware(), handlers.DeleteCover)
		users.PUT("/me", middleware.AuthMiddleware(), handlers.UpdateMe)
	}

		files := api.Group("/files")
		{
			files.GET("/:category/:hash", middleware.AuthMiddleware(), handlers.ServeFile)
		}

	// Push notification routes
		notify := api.Group("/notifications")
		{
			notify.POST("/register", middleware.AuthMiddleware(), handlers.RegisterDevice)
			notify.DELETE("/unregister", middleware.AuthMiddleware(), handlers.UnregisterDevice)
			notify.POST("/test", middleware.AuthMiddleware(), handlers.TestNotification)
		}

		// Case routes
		cases := api.Group("/cases")
		{
			cases.GET("", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.GetAllCases)
			cases.POST("", middleware.AuthMiddleware(), handlers.CreateCase)
			cases.GET("/analytics", middleware.AuthMiddleware(), handlers.GetCaseAnalytics)
		cases.GET("/:id", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), middleware.CanAccessCase, handlers.GetCaseByID)
		cases.PUT("/:id", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.UpdateCaseStatus)
		cases.POST("/:id/timeline", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.AddCaseTimeline)
		cases.GET("/:id/timeline", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.GetCaseTimeline)
		cases.POST("/:id/feedback", middleware.AuthMiddleware(), handlers.SubmitCaseFeedback)
		cases.POST("/:id/assign", middleware.AuthMiddleware(), middleware.IdempotencyMiddleware(), middleware.CanAccessCase, handlers.AssignCase)
		cases.GET("/:id/assignments", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.GetCaseAssignments)
		cases.POST("/:id/dispatch", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.DispatchCase)
		cases.POST("/:id/arrive", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.ArriveAtCase)
		cases.POST("/:id/progress", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.AddCaseProgress)
		cases.GET("/:id/progress", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.GetCaseProgress)
		cases.POST("/:id/submit-review", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.SubmitCaseForReview)
		cases.GET("/:id/review", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.GetCaseReview)
		cases.POST("/:id/review/request-changes", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.RequestCaseChanges)
		cases.POST("/:id/review/approve", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.ApproveCaseClosure)

		cases.POST("/:id/weekly-update", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.SubmitWeeklyCaseUpdate)
		cases.GET("/:id/weekly-updates", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.GetCaseWeeklyUpdates)
		cases.PUT("/:id/counter-statement", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.PutCounterStatement)
		cases.GET("/:id/counter-statement", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.GetCounterStatement)
		}
		// Election routes
		elections := api.Group("/elections")
		{
			elections.POST("/:id/vote", middleware.AuthMiddleware(), middleware.RateLimitVote(), middleware.IdempotencyMiddleware(), handlers.CastAdminVote)
			elections.POST("/:id/close", middleware.AuthMiddleware(), middleware.IdempotencyMiddleware(), handlers.CloseAdminElection)
			elections.GET("/:id/results", middleware.AuthMiddleware(), handlers.GetElectionResults)
			elections.POST("/seats/:id/fill", middleware.AuthMiddleware(), handlers.FillSeatVacancy)
		}
		// Revocation routes
		revocations := api.Group("/revocations")
		{
			revocations.POST("/:id/vote", middleware.AuthMiddleware(), middleware.RateLimitVote(), middleware.IdempotencyMiddleware(), handlers.CastRevocationVote)
			revocations.POST("/:id/close", middleware.AuthMiddleware(), middleware.IdempotencyMiddleware(), handlers.CloseRevocationCycle)
			revocations.GET("/:id", middleware.AuthMiddleware(), handlers.GetRevocationCycle)
		}

		// Geo routes (Wave 10.1c) — public reverse geocoding.
		geo := api.Group("/geo")
		{
			geo.GET("/reverse", middleware.RateLimitMap(), handlers.ReverseGeocode)
			geo.GET("/states", middleware.RateLimitMap(), handlers.ListStates)
			geo.GET("/lgas", middleware.RateLimitMap(), handlers.ListLGAs)
		}

		// Location routes
		location := api.Group("/location")
		{
			location.POST("", middleware.AuthMiddleware(), middleware.RateLimitMap(), handlers.UpdateMyLocation)
			location.GET("/sharing", middleware.AuthMiddleware(), handlers.GetLocationSharing)
			location.PUT("/sharing", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.UpdateLocationSharing)
		}

		// Evidence routes
		evidence := api.Group("/evidence")
		{
		evidence.POST("/upload", middleware.AuthMiddleware(), handlers.UploadEvidence)
		evidence.POST("/case/:caseId/file", middleware.AuthMiddleware(), middleware.UploadValidationMiddleware("evidence"), middleware.CanAccessCase, handlers.UploadEvidenceFile)
			evidence.POST("/case/:caseId/presign", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.PresignEvidenceUpload)
			evidence.POST("/case/:caseId/confirm", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.ConfirmEvidenceUpload)
		evidence.GET("/case/:caseId", middleware.AuthMiddleware(), middleware.CanAccessCase, handlers.GetEvidenceByCase)
			evidence.DELETE("/:id", middleware.AuthMiddleware(), handlers.DeleteEvidence)
			evidence.PATCH("/:id/verify", middleware.AuthMiddleware(), handlers.VerifyEvidence)
		}

		// Rating routes
		ratings := api.Group("/ratings")
		{
			ratings.POST("", middleware.AuthMiddleware(), handlers.SubmitRating)
			ratings.POST("/:id/flag", middleware.AuthMiddleware(), handlers.FlagRating)
		}

		// SOS routes
		sos := api.Group("/sos")
		{
			sos.POST("/send", middleware.AuthMiddleware(), handlers.SendSOSAlert)
			sos.GET("", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.GetSOSAlerts)
			sos.GET("/my", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.GetUserSOSAlerts)
			sos.GET("/:id", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.GetSOSAlertByID)
			sos.PUT("/:id/status", middleware.AuthMiddleware(), handlers.UpdateSOSAlertStatus)
		}

		// Suspect routes
		suspects := api.Group("/suspects")
		{
			suspects.GET("/me/cases", middleware.AuthMiddleware(), handlers.GetMySuspectCases)
		suspects.POST("", middleware.AuthMiddleware(), handlers.CreateSuspect)
			suspects.GET("", middleware.AuthMiddleware(), handlers.GetAllSuspects)
			suspects.GET("/:id", middleware.AuthMiddleware(), handlers.GetSuspectByID)
			suspects.PUT("/:id", middleware.AuthMiddleware(), handlers.UpdateSuspect)
			suspects.DELETE("/:id", middleware.AuthMiddleware(), handlers.DeleteSuspect)
			suspects.POST("/:id/sighting", middleware.AuthMiddleware(), handlers.ReportSighting)
			suspects.GET("/:id/sightings", middleware.AuthMiddleware(), handlers.GetSuspectSightings)
			suspects.GET("/:id/cases", middleware.AuthMiddleware(), handlers.GetSuspectCases)
			suspects.POST("/:id/associations", middleware.AuthMiddleware(), handlers.CreateSuspectAssociation)
			suspects.POST("/:id/expungement", middleware.AuthMiddleware(), handlers.RequestExpungement)
			suspects.GET("/:id/expungement", middleware.AuthMiddleware(), handlers.ListMyExpungementRequests)
		}

		// Expungement decision routes
		expungementGroup := api.Group("/expungement-requests")
		{
			expungementGroup.PUT("/:id/decision", middleware.AuthMiddleware(), handlers.DecideExpungement)
		}

		appeals := api.Group("/appeals")
		{
			appeals.POST("", middleware.AuthMiddleware(), handlers.FileAppeal)
			appeals.GET("", middleware.AuthMiddleware(), handlers.ListAppeals)
			appeals.GET("/:id", middleware.AuthMiddleware(), handlers.GetAppeal)
			appeals.PUT("/:id/decision", middleware.AuthMiddleware(), handlers.DecideAppeal)
		}

		// Transfer routes
		transfers := api.Group("/transfers")
		{
			transfers.POST("", middleware.AuthMiddleware(), handlers.RequestTransfer)
			transfers.POST("/:id/approve", middleware.AuthMiddleware(), handlers.ApproveTransfer)
			transfers.POST("/:id/reject", middleware.AuthMiddleware(), handlers.RejectTransfer)
			transfers.GET("", middleware.AuthMiddleware(), handlers.GetTransferRequests)
			transfers.GET("/:id/approvals", middleware.AuthMiddleware(), handlers.GetTransferApprovals)
		}

		// News routes
		news := api.Group("/news")
		{
			news.POST("", middleware.AuthMiddleware(), handlers.CreateNews)
			news.GET("", middleware.AuthMiddleware(), handlers.GetAllNews)
			news.GET("/:id", middleware.AuthMiddleware(), handlers.GetNewsByID)
		}

		// AI routes
		ai := api.Group("/ai")
		{
			ai.POST("/chatbot", middleware.AuthMiddleware(), handlers.AIChatbot)
			ai.POST("/analyze-image", middleware.AuthMiddleware(), handlers.AIAnalyzeImage)
			ai.POST("/analyze-location", middleware.AuthMiddleware(), middleware.RateLimitMap(), handlers.AIAnalyzeLocation)
			ai.POST("/map-insights", middleware.AuthMiddleware(), middleware.RateLimitMap(), handlers.AIGetMapInsights)
			ai.POST("/security-warning", middleware.AuthMiddleware(), handlers.AIGenerateSecurityWarning)
			ai.POST("/analyze-news", middleware.AuthMiddleware(), handlers.AIAnalyzeNews)
			ai.POST("/smart-tips", middleware.AuthMiddleware(), handlers.AIGetSmartTips)
			ai.POST("/predict-hotspots", middleware.AuthMiddleware(), middleware.RateLimitMap(), handlers.AIPredictHotspots)
		}

		// Bank Account routes
		bank := api.Group("/bank")
		{
			bank.POST("/accounts", middleware.AuthMiddleware(), handlers.AddBankAccount)
			bank.GET("/:id/accounts", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.GetBankAccounts)
			bank.PUT("/accounts/:id", middleware.AuthMiddleware(), handlers.UpdateBankAccount)
			bank.DELETE("/accounts/:id", middleware.AuthMiddleware(), handlers.DeleteBankAccount)
			bank.PATCH("/accounts/:id/public", middleware.AuthMiddleware(), handlers.ToggleBankAccountPublic)
			bank.GET("/units/:id/ledger", middleware.AuthMiddleware(), handlers.GetUnitLedger)
			bank.POST("/donations", middleware.AuthMiddleware(), handlers.RecordDonation)
			bank.POST("/donations/:id/confirm", middleware.AuthMiddleware(), handlers.ConfirmDonation)
			bank.GET("/:id/donations", middleware.AuthMiddleware(), handlers.GetDonations)
		}

		// Finance routes
		finance := api.Group("/finance")
		{
			finance.POST("/transactions", middleware.AuthMiddleware(), handlers.CreateTransaction)
			finance.POST("/transactions/:id/approve", middleware.AuthMiddleware(), handlers.ApproveTransaction)
			finance.POST("/transactions/:id/reject", middleware.AuthMiddleware(), handlers.RejectTransaction)
			finance.GET("/units/:id/transactions", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.GetTransactions)
			finance.GET("/transactions/:id", middleware.AuthMiddleware(), handlers.GetTransactionByID)
			finance.GET("/units/:id/summary", middleware.AuthMiddleware(), handlers.GetTransactionSummary)
			finance.POST("/budgets", middleware.AuthMiddleware(), handlers.CreateBudget)
			finance.GET("/units/:id/budgets", middleware.AuthMiddleware(), handlers.GetBudgets)
			finance.POST("/reports", middleware.AuthMiddleware(), handlers.GenerateFinancialReport)
			finance.GET("/units/:id/reports", middleware.AuthMiddleware(), handlers.GetFinancialReports)
		}

		// Community routes
		community := api.Group("/community")
		{
			community.POST("/posts", middleware.AuthMiddleware(), handlers.CreateForumPost)
			community.GET("/posts", middleware.AuthMiddleware(), handlers.GetForumPosts)
			community.GET("/posts/:id", middleware.AuthMiddleware(), handlers.GetForumPostByID)
			community.POST("/replies", middleware.AuthMiddleware(), handlers.CreateForumReply)
			community.POST("/announcements", middleware.AuthMiddleware(), handlers.CreateCommunityAnnouncement)
			community.GET("/announcements", middleware.AuthMiddleware(), handlers.GetCommunityAnnouncements)
			community.POST("/events", middleware.AuthMiddleware(), handlers.CreateCommunityEvent)
			community.GET("/events", middleware.AuthMiddleware(), handlers.GetCommunityEvents)
			community.POST("/events/:id/rsvp", middleware.AuthMiddleware(), handlers.RSVPToEvent)
		}

		// Audit routes
		audit := api.Group("/audit")
		{
			audit.POST("/activity", middleware.AuthMiddleware(), handlers.LogActivity)
			audit.GET("/activities", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.GetActivityLogs)
			audit.GET("/logs", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.GetAuditLogs)
			audit.POST("/logs", middleware.AuthMiddleware(), handlers.CreateAuditLog)
			audit.GET("/health", middleware.AuthMiddleware(), handlers.GetSystemHealth)
			audit.POST("/health", middleware.AuthMiddleware(), handlers.UpdateSystemHealth)
			audit.GET("/notifications", middleware.AuthMiddleware(), handlers.GetNotificationLogs)
		}

		// Alert routes
		// Alert routes
		alerts := api.Group("/alerts")
		{
			alerts.GET("/news", middleware.AuthMiddleware(), handlers.GetNewsAlerts)

			alerts.POST("", middleware.AuthMiddleware(), handlers.CreateCommunityAlert)
			alerts.GET("", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.GetCommunityAlerts)

			alerts.POST("/subscribe", middleware.AuthMiddleware(), handlers.SubscribeToAlerts)
			alerts.GET("/subscriptions", middleware.AuthMiddleware(), handlers.GetAlertSubscriptions)

			alerts.GET("/:id", middleware.AuthMiddleware(), handlers.GetAlertByID)
			alerts.POST("/:id/confirm", middleware.AuthMiddleware(), handlers.ConfirmAlert)
		}

		// Settings routes
		settings := api.Group("/settings")
		{
			settings.GET("/public", handlers.GetPublicSettings)

			settings.GET("/templates/:name", middleware.AuthMiddleware(), handlers.GetEmailTemplate)
			settings.PUT("/templates/:name", middleware.AuthMiddleware(), handlers.UpdateEmailTemplate)

			settings.POST("/exports", middleware.AuthMiddleware(), handlers.CreateDataExport)
			settings.GET("/exports", middleware.AuthMiddleware(), handlers.GetDataExports)

			settings.GET("/onboarding", middleware.AuthMiddleware(), handlers.GetUserOnboarding)
			settings.PUT("/onboarding", middleware.AuthMiddleware(), handlers.UpdateUserOnboarding)

			settings.GET("/:key", middleware.AuthMiddleware(), handlers.GetSystemSetting)
			settings.PUT("/:key", middleware.AuthMiddleware(), handlers.UpdateSystemSetting)
		}

		// Mobile API routes
		mobile := api.Group("/mobile")
		{
			mobile.GET("/config", middleware.AuthMiddleware(), handlers.MobileAppConfig)
			mobile.GET("/dashboard", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.MobileDashboard)
			mobile.GET("/notifications", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.MobileNotifications)
			mobile.PUT("/notifications/:id/read", middleware.AuthMiddleware(), handlers.MobileMarkNotificationRead)
			mobile.PUT("/notifications/read-all", middleware.AuthMiddleware(), handlers.MobileMarkAllRead)
			mobile.GET("/sync", middleware.AuthMiddleware(), middleware.RateLimitGeneral(), handlers.MobileSync)
			mobile.POST("/crash-report", middleware.AuthMiddleware(), handlers.MobileCrashReport)
		}

		// Video Analytics routes
		video := api.Group("/video")
		{
			video.POST("/cameras", middleware.AuthMiddleware(), handlers.AddCamera)
			video.GET("/units/:id/cameras", middleware.AuthMiddleware(), handlers.GetCameras)
			video.POST("/alerts", middleware.AuthMiddleware(), handlers.GenerateVideoAlert)
			video.GET("/alerts", middleware.AuthMiddleware(), handlers.GetVideoAlerts)
			video.PUT("/alerts/:id/review", middleware.AuthMiddleware(), handlers.ReviewVideoAlert)
			video.POST("/social/monitor", middleware.AuthMiddleware(), handlers.MonitorSocialMedia)
			video.GET("/social/posts", middleware.AuthMiddleware(), handlers.GetSocialMediaPosts)
		}

		// Communication routes
		commGroup := api.Group("/communication")
		{
			commGroup.POST("/rooms", middleware.AuthMiddleware(), handlers.CreateRoom)
			commGroup.GET("/units/:id/rooms", middleware.AuthMiddleware(), handlers.GetRooms)
			commGroup.POST("/messages", middleware.AuthMiddleware(), handlers.SendMessage)
			commGroup.GET("/rooms/:roomId/messages", middleware.AuthMiddleware(), handlers.GetMessages)
			commGroup.POST("/calls", middleware.AuthMiddleware(), handlers.InitiateCall)
			commGroup.PUT("/calls/:id/end", middleware.AuthMiddleware(), handlers.EndCall)
			commGroup.POST("/sync", middleware.AuthMiddleware(), handlers.SyncMessages)
			commGroup.GET("/rooms/:roomId/sync-status", middleware.AuthMiddleware(), handlers.GetSyncStatus)
		}

		// SMS routes
		sms := api.Group("/sms")
		{
			sms.POST("/incoming", middleware.SMSWebhookSignature(), handlers.HandleIncomingSMS)
			sms.POST("/ussd", middleware.SMSWebhookSignature(), handlers.HandleUSSD)
		}

		// Peacebuilding routes
		peace := api.Group("/peacebuilding")
		{
			peace.POST("/committees", middleware.AuthMiddleware(), handlers.CreatePeaceCommittee)
			peace.GET("/units/:id/committees", middleware.AuthMiddleware(), handlers.GetPeaceCommittees)
			peace.POST("/committees/:id/members", middleware.AuthMiddleware(), handlers.AddCommitteeMember)
			peace.POST("/conflicts", middleware.AuthMiddleware(), handlers.CreateConflictResolution)
			peace.GET("/units/:id/conflicts", middleware.AuthMiddleware(), handlers.GetConflictResolutions)
			peace.PUT("/conflicts/:id", middleware.AuthMiddleware(), handlers.UpdateConflictResolution)
			peace.GET("/units/:id/peace-metrics", middleware.AuthMiddleware(), handlers.GetPeaceMetrics)
			peace.GET("/units/:id/trust-scores", middleware.AuthMiddleware(), handlers.GetTrustScores)
			peace.POST("/trust-scores", middleware.AuthMiddleware(), handlers.UpdateTrustScore)
		}

		// Super Admin routes
		superAdmin := api.Group("/admin")
		superAdmin.Use(middleware.AuthMiddleware(), handlers.SuperAdminMiddleware())
		{
			superAdmin.GET("/users", handlers.GetAllUsers)
			superAdmin.GET("/users/:id", handlers.GetUserByID)
			superAdmin.PUT("/users/:id/role", handlers.UpdateUserRole)
			superAdmin.POST("/users/:id/suspend", handlers.SuspendUser)
			superAdmin.POST("/users/:id/activate", handlers.ActivateUser)
			superAdmin.POST("/users/:id/impersonate", handlers.ImpersonateUser)
			superAdmin.PUT("/users/:id/age-override", handlers.AgeOverride)
			superAdmin.POST("/stop-impersonate", handlers.StopImpersonation)
			superAdmin.GET("/stats", handlers.GetSystemStats)
			superAdmin.GET("/platform-donations", handlers.ListPlatformDonations)
			superAdmin.POST("/platform-donations/:id/confirm", handlers.ConfirmPlatformDonation)
		}
	}

	// WebSocket route (protected)
	router.GET("/ws", middleware.AuthMiddleware(), handlers.HandleWebSocket)

	// Metrics endpoint — authed, super-admin only. Exposes in-process
	// counters (request rate, error rate, rolling latency). A future wave
	// can swap the JSON serializer for a Prometheus text encoder without
	// changing this route registration.
	router.GET("/metrics", middleware.AuthMiddleware(), handlers.SuperAdminMiddleware(), handlers.GetMetrics)
}
