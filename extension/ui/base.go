package ui

import (
	"encoding/json"
	"fmt"
)

// Field is an interface that all specific field types implement.
type Field interface {
	GetType() string
}

// GenericField holds common field properties.
const (
	FieldSelect         string = "Select"
	FieldEmail          string = "Email"
	FieldInput          string = "Input"
	FieldPassword       string = "Password"
	FieldTextArea       string = "TextArea"
	FieldSwitch         string = "Switch"
	FieldCheckbox       string = "Checkbox"
	FieldRadioButton    string = "RadioButton"
	FieldConsole        string = "Console"
	FieldButton         string = "Button"
	ValidatorDigitsOnly string = "digitsOnly"

	ButtonSubmit string = "Submit"
	ButtonCancel string = "Cancel"

	ButtonDialogClose string = "CloseDialog"
	ButtonDialogOk    string = "OkDialog"
)

// FormField extends GenericField with additional common properties.
type FormField struct {
	Key         string       `json:"key"`
	Type        string       `json:"type"`
	Label       string       `json:"label,omitempty"`
	LabelHidden bool         `json:"labelHidden"`
	Required    bool         `json:"required,omitempty"`
	Placeholder string       `json:"placeholder,omitempty"`
	Readonly    bool         `json:"readonly,omitempty"`
	Value       string       `json:"value"`
	Validator   string       `json:"validator,omitempty"`
	Items       []SelectItem `json:"items,omitempty"`
	Lines       int          `json:"lines,omitempty"`
}

// GetType returns the type of the field.
func (gf FormField) GetType() string {
	return gf.Type
}

type InputField struct {
	FormField
	Validator string `json:"validator,omitempty"`
}

type SelectItem struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Form struct {
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Fields      [][]FormField `json:"fields"`
	// Buttons     []string      `json:"buttons"`
}

func (f *Form) ToJSON() string {
	formJson, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		fmt.Println("Error encoding to JSON:", err)
		return ""
	}

	return (string(formJson))
}

// UnmarshalJSON custom unmarshals JSON data into a Form.
func (f *Form) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}

	return nil
}
