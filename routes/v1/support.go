package v1

import (
	"api/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SupportRequest struct {
	Name      string `json:"name" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	IssueType string `json:"issueType" binding:"required"`
	Subject   string `json:"subject" binding:"required"`
	Message   string `json:"message" binding:"required"`
}

// @Summary Submit a support request
// @Description Sends a support email with the user's request
// @Tags Support
// @Accept json
// @Produce json
// @Param request body SupportRequest true "Support request details"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /support [post]
func submitSupportRequest(c *gin.Context) {
	var request SupportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Send the support email
	emailService := services.NewEmailService()
	err := emailService.SendSupportEmail(request.Name, request.Email, request.IssueType, request.Subject, request.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send support email: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Support request submitted successfully",
	})
}

func RegisterSupportRoutes(r *gin.RouterGroup) {
	r.POST("/support", submitSupportRequest)
}