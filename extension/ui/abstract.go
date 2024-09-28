package ui

// // Field is an interface that all specific field types implement.
// type Field interface {
// 	GetType() string
// }

// // GenericField holds common field properties.
// const (
// 	Select      string = "Select"
// 	Email       string = "Email"
// 	Input       string = "Input"
// 	Password    string = "Password"
// 	TextArea    string = "TextArea"
// 	Switch      string = "Switch"
// 	Checkbox    string = "Checkbox"
// 	RadioButton string = "RadioButton"
// 	DigitsOnly  string = "digitsOnly"
// )

// // FormField extends GenericField with additional common properties.
// type FormField struct {
// 	Key              string       `json:"key"`
// 	Type             string       `json:"type"`
// 	Label            string       `json:"label,omitempty"`
// 	LabelHidden      bool         `json:"labelHidden"`
// 	Required         bool         `json:"required,omitempty"`
// 	Placeholder      string       `json:"placeholder,omitempty"`
// 	Readonly         bool         `json:"readonly,omitempty"`
// 	Value            string       `json:"value"`
// 	Validator        string       `json:"validator,omitempty"`
// 	Items            []SelectItem `json:"items,omitempty"`
// 	Lines            int          `json:"lines,omitempty"`
// 	VerticalScroll   bool         `json:"verticalScroll,omitempty"`
// 	HorizontalScroll bool         `json:"horizontalScroll,omitempty"`
// 	Monospace        bool         `json:"monospace,omitempty"`
// }

// // GetType returns the type of the field.
// func (gf FormField) GetType() string {
// 	return gf.Type
// }
