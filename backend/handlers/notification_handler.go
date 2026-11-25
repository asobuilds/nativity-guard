package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
)

func dispatchWebPush(userID uuid.UUID, title, message string) {
	var subs []models.PushSubscription
	if err := config.DB.Where("user_id = ?", userID).Find(&subs).Error; err != nil || len(subs) == 0 {
		return
	}

	vapidPublicKey := os.Getenv("VAPID_PUBLIC_KEY")
	vapidPrivateKey := os.Getenv("VAPID_PRIVATE_KEY")
	vapidSubscriber := os.Getenv("VAPID_SUBSCRIBER")
	if vapidPublicKey == "" || vapidPrivateKey == "" {
		return
	}

	suber := vapidSubscriber
	if suber == "" {
		suber = "mailto:support@nativityguard.org"
	}

	payloadBytes, _ := json.Marshal(map[string]string{
		"title":   title,
		"message": message,
	})

	for _, sub := range subs {
		s := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}

		_, err := webpush.SendNotification(payloadBytes, s, &webpush.Options{
			Subscriber:      suber,
			VAPIDPublicKey:  vapidPublicKey,
			VAPIDPrivateKey: vapidPrivateKey,
			TTL:             30,
		})
		if err != nil {
			log.Printf("[WebPush] failed to send notification to %s: %v", sub.Endpoint, err)
		}
	}
}

func SendNotification(c *gin.Context) {
	var input struct {
		UserID  string `json:"userId" binding:"required"`
		Title   string `json:"title" binding:"required"`
		Message string `json:"message" binding:"required"`
		Type    string `json:"type"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	notification := models.Notification{
		UserID:  userID,
		Title:   input.Title,
		Message: input.Message,
		Type:    input.Type,
		Status:  "unread",
	}

	if err := config.DB.Create(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send notification"})
		return
	}

	go dispatchWebPush(notification.UserID, notification.Title, notification.Message)

	c.JSON(http.StatusCreated, gin.H{
		"message":      "Notification sent successfully",
		"notification": notification,
	})
}

func SubscribePush(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var input struct {
		Endpoint string `json:"endpoint" binding:"required"`
		Keys     struct {
			P256dh string `json:"p256dh" binding:"required"`
			Auth   string `json:"auth" binding:"required"`
		} `json:"keys" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub := models.PushSubscription{
		UserID:   userObj.ID,
		Endpoint: input.Endpoint,
		P256dh:   input.Keys.P256dh,
		Auth:     input.Keys.Auth,
	}

	if err := config.DB.Where("endpoint = ?", input.Endpoint).Assign(models.PushSubscription{
		UserID: userObj.ID,
		P256dh: input.Keys.P256dh,
		Auth:   input.Keys.Auth,
	}).FirstOrCreate(&sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save push subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscribed to push notifications successfully"})
}

func GetUserNotifications(c *gin.Context) {
	userID := c.Param("userId")
	id, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var notifications []models.Notification
	if err := config.DB.Where("user_id = ?", id).Order("created_at desc").Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
	})
}

func MarkNotificationAsRead(c *gin.Context) {
	id := c.Param("id")
	notificationID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	var notification models.Notification
	if err := config.DB.First(&notification, "id = ?", notificationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	notification.Status = "read"
	if err := config.DB.Save(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Notification marked as read",
	})
}

func DeleteNotification(c *gin.Context) {
	id := c.Param("id")
	notificationID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	if err := config.DB.Delete(&models.Notification{}, "id = ?", notificationID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Notification deleted successfully",
	})
}
