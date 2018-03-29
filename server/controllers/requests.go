package controllers

type idRequest struct {
	ID uint `json:"id"`
}

type email struct {
	Email string `json:"email" validate:"required,email"`
}
