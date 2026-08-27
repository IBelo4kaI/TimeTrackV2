package notificationtemplate

type CreateNotificationTemplateRequest struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

type UpdateNotificationTemplateRequest struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Message string `json:"message"`
}
