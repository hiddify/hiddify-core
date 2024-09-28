package ui

// import (
// 	"encoding/json"
// 	"fmt"
// )

// // InputField represents a text input field.
// type InputField struct {
// 	FormField
// 	Validator string `json:"validator,omitempty"`

// }

// // // NewInputField creates a new InputField.
// // func NewInputField(key, label, placeholder string, required bool, value string) InputField {
// // 	return InputField{
// // 		FormField: FormField{
// // 			GenericField: GenericField{
// // 				Key:   key,
// // 				Type:  "Input",
// // 				Label: label,
// // 			},
// // 			Placeholder: placeholder,
// // 			Required:    required,
// // 			Value:       value,
// // 		},
// // 	}
// // }

// // // PasswordField represents a password field.
// // type PasswordField struct {
// // 	FormField
// // }

// // // NewPasswordField creates a new PasswordField.
// // func NewPasswordField(key, label string, required bool, value string) PasswordField {
// // 	return PasswordField{
// // 		FormField: FormField{
// // 			GenericField: GenericField{
// // 				Key:   key,
// // 				Type:  "Password",
// // 				Label: label,
// // 			},
// // 			Required: required,
// // 			Value:    value,
// // 		},
// // 	}
// // }

// // // EmailField represents an email field.
// // type EmailField struct {
// // 	FormField
// // }

// // // NewEmailField creates a new EmailField.
// // func NewEmailField(key, label, placeholder string, required bool, value string) EmailField {
// // 	return EmailField{
// // 		FormField: FormField{
// // 			GenericField: GenericField{
// // 				Key:   key,
// // 				Type:  "Email",
// // 				Label: label,
// // 			},
// // 			Placeholder: placeholder,
// // 			Required:    required,
// // 			Value:       value,
// // 		},
// // 	}
// // }

// // // TextAreaField represents a multi-line text area field.
// // type TextAreaField struct {
// // 	FormField
// // }

// // // NewTextAreaField creates a new TextAreaField.
// // func NewTextAreaField(key, label, placeholder string, required bool, value string) TextAreaField {
// // 	return TextAreaField{
// // 		FormField: FormField{
// // 			GenericField: GenericField{
// // 				Key:   key,
// // 				Type:  "TextArea",
// // 				Label: label,
// // 			},
// // 			Placeholder: placeholder,
// // 			Required:    required,
// // 			Value:       value,
// // 		},
// // 	}
// // }

// // // SelectField represents a dropdown selection field.
// // type SelectField struct {
// // 	FormField
// // 	Items []SelectItem `json:"items"`
// // }

// // // SelectItem represents an item in a dropdown.
// type SelectItem struct {
// 	Label string `json:"label"`
// 	Value string `json:"value"`
// }

// // // NewSelectField creates a new SelectField.
// // func NewSelectField(key, label, value string, items []SelectItem) SelectField {
// // 	return SelectField{
// // 		FormField: FormField{
// // 			GenericField: GenericField{
// // 				Key:   key,
// // 				Type:  "Select",
// // 				Label: label,
// // 			},
// // 			Value: value,
// // 		},
// // 		Items: items,
// // 	}
// // }

// // Form represents a collection of fields with metadata.
// type Form struct {
// 	Title       string      `json:"title"`
// 	Description string      `json:"description"`
// 	Fields      []FormField `json:"fields"`
// }

// func (f *Form) ToJSON() string {
// 	formJson, err := json.MarshalIndent(f, "", "  ")
// 	if err != nil {
// 		fmt.Println("Error encoding to JSON:", err)
// 		return ""
// 	}
// 	return (string(formJson))
// }

// // UnmarshalJSON custom unmarshals JSON data into a Form.
// func (f *Form) UnmarshalJSON(data []byte) error {
// 	if err := json.Unmarshal(data, &f); err != nil {
// 		return err
// 	}

// 	// f.Title = raw.Title
// 	// f.Description = raw.Description

// 	// for _, fieldData := range raw.Fields {
// 	// 	var base FormField
// 	// 	if err := json.Unmarshal(fieldData, &base); err != nil {
// 	// 		return err
// 	// 	}

// 	// 	var field Field
// 	// 	switch base.Type {
// 	// 	case "Input":
// 	// 		var inputField InputField
// 	// 		if err := json.Unmarshal(fieldData, &inputField); err != nil {
// 	// 			return err
// 	// 		}
// 	// 		field = inputField
// 	// 	case "Password":
// 	// 		var passwordField PasswordField
// 	// 		if err := json.Unmarshal(fieldData, &passwordField); err != nil {
// 	// 			return err
// 	// 		}
// 	// 		field = passwordField
// 	// 	case "Email":
// 	// 		var emailField EmailField
// 	// 		if err := json.Unmarshal(fieldData, &emailField); err != nil {
// 	// 			return err
// 	// 		}
// 	// 		field = emailField
// 	// 	case "TextArea":
// 	// 		var textAreaField TextAreaField
// 	// 		if err := json.Unmarshal(fieldData, &textAreaField); err != nil {
// 	// 			return err
// 	// 		}
// 	// 		field = textAreaField
// 	// 	case "Select":
// 	// 		var selectField SelectField
// 	// 		if err := json.Unmarshal(fieldData, &selectField); err != nil {
// 	// 			return err
// 	// 		}
// 	// 		field = selectField
// 	// 	case "Content":
// 	// 		var contentField ContentField
// 	// 		if err := json.Unmarshal(fieldData, &contentField); err != nil {
// 	// 			return err
// 	// 		}
// 	// 		field = contentField
// 	// 	default:
// 	// 		return fmt.Errorf("unsupported field type: %s", base.Type)
// 	// 	}

// 	// 	f.Fields = append(f.Fields, field)
// 	// }

// 	return nil
// }

// // func main() {
// // 	// Example form JSON
// // 	formJSON := `{
// // 		"title": "Form Example",
// // 		"description": "",
// // 		"fields": [
// // 			{
// // 				"key": "inputKey",
// // 				"type": "Input",
// // 				"label": "Hi Group",
// // 				"placeholder": "Hi Group flutter",
// // 				"required": true,
// // 				"value": "D"
// // 			},
// // 			{
// // 				"key": "passwordKey",
// // 				"type": "Password",
// // 				"label": "Password",
// // 				"required": true,
// // 				"value": "secret"
// // 			},
// // 			{
// // 				"key": "emailKey",
// // 				"type": "Email",
// // 				"label": "Email Label",
// // 				"placeholder": "Enter your email",
// // 				"required": true,
// // 				"value": "example@example.com"
// // 			}
// // 		]
// // 	}`

// // 	var form Form

// // 	// Decode the form JSON
// // 	if err := json.Unmarshal([]byte(formJSON), &form); err != nil {
// // 		fmt.Println("Error decoding form:", err)
// // 		return
// // 	}

// // 	// Print decoded form fields
// // 	fmt.Println("Form Title:", form.Title)
// // 	for i, field := range form.Fields {
// // 		fmt.Printf("Field %d: %T\n", i+1, field)
// // 	}
// // }
