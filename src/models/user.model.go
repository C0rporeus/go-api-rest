package userModel

type User struct {
	UserId   string `json:"userId"`
	Email    string `json:"email"`
	Password string `json:"password"`
	UserName string `json:"username"`
}
