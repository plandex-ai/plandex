pdx-1: package auth 
pdx-2: 
pdx-3: // User represents a user entity
pdx-4: 
pdx-5: type User struct {
pdx-6: 	ID   string
pdx-7: 	Name string
pdx-8: }
pdx-9: 
pdx-10: // AuthService interface for authentication
pdx-11: 
pdx-12: type AuthService interface {
pdx-13: 	Login(username, password string) (*User, error)
pdx-14: 	Register(user User) error
pdx-15: }
