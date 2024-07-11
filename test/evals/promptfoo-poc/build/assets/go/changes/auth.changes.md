// Implementing AuthService using mock data for testing purposes

type MockAuthService struct{}

// Mock Login function

type MockAuthService func(username, password string) (*User, error) {

return &User{ID: "1", Name: "Test User"}, nil

}

// Mock Register function

type MockAuthService func(user User) error {

return nil

}

// Added ValidateToken middleware function

func ValidateToken(token string) bool {

// This is a mock implementation

return token == "valid-token"

}

