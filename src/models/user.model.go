package userModel

// User representa un usuario en el sistema
// @Description Modelo de usuario.
type User struct {
	UserId   string `json:"userId"`
	Email    string `json:"email"`
	Password string `json:"password"`
	UserName string `json:"username"`
}
