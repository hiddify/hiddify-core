package ui

// import (
// 	"encoding/json"
// 	"testing"
// )

// // Test UnmarshalJSON for different field types
// func TestFormUnmarshalJSON(t *testing.T) {
// 	formJSON := `{
// 		"title": "Form Example",
// 		"description": "This is a sample form.",
// 		"fields": [
// 			{
// 				"key": "inputKey",
// 				"type": "Input",
// 				"label": "Hi Group",
// 				"placeholder": "Hi Group flutter",
// 				"required": true,
// 				"value": "D"
// 			},
// 			{
// 				"key": "passwordKey",
// 				"type": "Password",
// 				"label": "Password",
// 				"required": true,
// 				"value": "secret"
// 			},
// 			{
// 				"key": "emailKey",
// 				"type": "Email",
// 				"label": "Email Label",
// 				"placeholder": "Enter your email",
// 				"required": true,
// 				"value": "example@example.com"
// 			}
// 		]
// 	}`

// 	var form Form
// 	err := json.Unmarshal([]byte(formJSON), &form)
// 	if err != nil {
// 		t.Fatalf("Error unmarshaling form JSON: %v", err)
// 	}

// 	if form.Title != "Form Example" {
// 		t.Errorf("Expected Title to be 'Form Example', got '%s'", form.Title)
// 	}
// 	if form.Description != "This is a sample form." {
// 		t.Errorf("Expected Description to be 'This is a sample form.', got '%s'", form.Description)
// 	}

// 	if len(form.Fields) != 3 {
// 		t.Fatalf("Expected 3 fields, got %d", len(form.Fields))
// 	}

// 	for i, field := range form.Fields {
// 		switch f := field.(type) {
// 		case InputField:
// 			if f.Type != "Input" {
// 				t.Errorf("Field %d: Expected Type to be 'Input', got '%s'", i+1, f.Type)
// 			}
// 		case PasswordField:
// 			if f.Type != "Password" {
// 				t.Errorf("Field %d: Expected Type to be 'Password', got '%s'", i+1, f.Type)
// 			}
// 		case EmailField:
// 			if f.Type != "Email" {
// 				t.Errorf("Field %d: Expected Type to be 'Email', got '%s'", i+1, f.Type)
// 			}
// 		default:
// 			t.Errorf("Field %d: Unexpected field type %T", i+1, f)
// 		}
// 	}
// }
