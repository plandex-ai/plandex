package syntax

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		want     *FileMap
	}{
		{
			name:     "simple go file",
			filename: "test.go",
			content: `package main

import "fmt"

// User represents a system user
type User struct {
    Name string
    Age  int
}

// GetName returns the user's name
func (u *User) GetName() string {
    return u.Name
}`,
			want: &FileMap{
				Definitions: []Definition{
					{
						Type:      "type_definition",
						Signature: "type User struct {\n    Name string\n    Age  int\n}",
						Comments:  []string{"// User represents a system user"},
						Line:      5,
						Children: []Definition{
							{
								Type:      "method_definition",
								Signature: "func (u *User) GetName() string",
								Comments:  []string{"// GetName returns the user's name"},
								Line:      11,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapFile(context.Background(), tt.filename, []byte(tt.content))
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
