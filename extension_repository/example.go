package extension_repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	ex "github.com/hiddify/hiddify-core/extension"
	ui "github.com/hiddify/hiddify-core/extension/ui_elements"
)

// Field name constants
const (
	CountKey    = "countKey"
	InputKey    = "inputKey"
	PasswordKey = "passwordKey"
	EmailKey    = "emailKey"
	SelectKey   = "selectKey"
	TextAreaKey = "textareaKey"
	SwitchKey   = "switchKey"
	CheckboxKey = "checkboxKey"
	RadioboxKey = "radioboxKey"
	ContentKey  = "contentKey"
)

type ExampleExtension struct {
	ex.BaseExtension
	cancel    context.CancelFunc
	input     string
	password  string
	email     string
	selected  bool
	textarea  string
	switchVal bool
	// checkbox  string
	radiobox string
	content  string

	count int
}

func NewExampleExtension() ex.Extension {
	return &ExampleExtension{
		input:     "default",
		password:  "123456",
		email:     "app@hiddify.com",
		selected:  false,
		textarea:  "area",
		switchVal: true,
		// checkbox:  "B",
		radiobox: "A",
		content:  "Welcome to Example Extension",
		count:    10,
	}
}

func (e *ExampleExtension) GetId() string {
	return "example"
}

func (e *ExampleExtension) GetTitle() string {
	return "Example Extension"
}

func (e *ExampleExtension) GetDescription() string {
	return "This is a sample extension."
}

func (e *ExampleExtension) GetUI() ui.Form {
	// e.setFormData(data)
	return e.buildForm()
}

func (e *ExampleExtension) setFormData(data map[string]string) error {
	if val, ok := data[CountKey]; ok {
		if intValue, err := strconv.Atoi(val); err == nil {
			e.count = intValue
		} else {
			return err
		}
	}
	if val, ok := data[InputKey]; ok {
		e.input = val
	}
	if val, ok := data[PasswordKey]; ok {
		e.password = val
	}
	if val, ok := data[EmailKey]; ok {
		e.email = val
	}
	if val, ok := data[SelectKey]; ok {
		if selectedValue, err := strconv.ParseBool(val); err == nil {
			e.selected = selectedValue
		} else {
			return err
		}
	}
	if val, ok := data[TextAreaKey]; ok {
		e.textarea = val
	}
	if val, ok := data[SwitchKey]; ok {
		if selectedValue, err := strconv.ParseBool(val); err == nil {
			e.switchVal = selectedValue
		} else {
			return err
		}
	}
	// if val, ok := data[CheckboxKey]; ok {
	// 	e.checkbox = val
	// }
	if val, ok := data[ContentKey]; ok {
		e.content = val
	}
	if val, ok := data[RadioboxKey]; ok {
		e.radiobox = val
	}
	return nil
}

func (e *ExampleExtension) buildForm() ui.Form {
	return ui.Form{
		Title:       "Example Form",
		Description: "This is a sample form.",
		ButtonMode:  ui.Button_SubmitCancel,
		Fields: []ui.FormField{
			{
				Type:        ui.FieldInput,
				Key:         CountKey,
				Label:       "Count",
				Placeholder: "This will be the count",
				Required:    true,
				Value:       fmt.Sprintf("%d", e.count),
				Validator:   ui.ValidatorDigitsOnly,
			},
			{
				Type:        ui.FieldInput,
				Key:         InputKey,
				Label:       "Hi Group",
				Placeholder: "Hi Group flutter",
				Required:    true,
				Value:       e.input,
			},
			{
				Type:     ui.FieldPassword,
				Key:      PasswordKey,
				Label:    "Password",
				Required: true,
				Value:    e.password,
			},
			{
				Type:        ui.FieldEmail,
				Key:         EmailKey,
				Label:       "Email Label",
				Placeholder: "Enter your email",
				Required:    true,
				Value:       e.email,
			},
			{
				Type:  ui.FieldSwitch,
				Key:   SelectKey,
				Label: "Select Label",
				Value: strconv.FormatBool(e.selected),
			},
			{
				Type:        ui.FieldTextArea,
				Key:         TextAreaKey,
				Label:       "TextArea Label",
				Placeholder: "Enter your text",
				Required:    true,
				Value:       e.textarea,
			},
			{
				Type:  ui.FieldSwitch,
				Key:   SwitchKey,
				Label: "Switch Label",
				Value: strconv.FormatBool(e.switchVal),
			},
			// {
			// 	Type:     ui.Checkbox,
			// 	Key:      CheckboxKey,
			// 	Label:    "Checkbox Label",
			// 	Required: true,
			// 	Value:    e.checkbox,
			// 	Items: []ui.SelectItem{
			// 		{
			// 			Label: "A",
			// 			Value: "A",
			// 		},
			// 		{
			// 			Label: "B",
			// 			Value: "B",
			// 		},
			// 	},
			// },
			{
				Type:     ui.FieldRadioButton,
				Key:      RadioboxKey,
				Label:    "Radio Label",
				Required: true,
				Value:    e.radiobox,
				Items: []ui.SelectItem{
					{
						Label: "A",
						Value: "A",
					},
					{
						Label: "B",
						Value: "B",
					},
				},
			},
			{
				Type:             ui.FieldTextArea,
				Readonly:         true,
				Key:              ContentKey,
				Label:            "Content",
				Value:            e.content,
				Lines:            10,
				Monospace:        true,
				HorizontalScroll: true,
				VerticalScroll:   true,
			},
		},
	}
}

func (e *ExampleExtension) backgroundTask(ctx context.Context) {
	i := 1
	for {
		select {
		case <-ctx.Done():
			e.content = strconv.Itoa(i) + " Background task stop...\n" + e.content
			e.UpdateUI(e.buildForm())

			fmt.Println("Background task stopped")
			return
		case <-time.After(1000 * time.Millisecond):
			txt := strconv.Itoa(i) + " Background task working..."
			e.content = txt + "\n" + e.content
			e.UpdateUI(e.buildForm())
			fmt.Println(txt)
		}
		i++
	}
}

func (e *ExampleExtension) SubmitData(data map[string]string) error {
	err := e.setFormData(data)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	go e.backgroundTask(ctx)

	return nil
}

func (e *ExampleExtension) Cancel() error {
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	return nil
}

func (e *ExampleExtension) Stop() error {
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	return nil
}

func init() {
	ex.RegisterExtension("com.example.extension", NewExampleExtension())
}
