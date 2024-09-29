
const ansi_up = new AnsiUp({
    escape_html: false,

});


function renderForm(json, dialog, submitAction, cancelAction, stopAction) {
    const container = document.getElementById(`extension-page-container${dialog}`);
    const formId = `dynamicForm${json.id}${dialog}`;

    const existingForm = document.getElementById(formId);
    if (existingForm) {
        existingForm.remove();
    }
    const form = document.createElement('form');
    container.appendChild(form);
    form.id = formId;

    if (dialog === "dialog") {
        document.getElementById("modalLabel").textContent = json.title;
    } else {
        const titleElement = createTitleElement(json);
        if (stopAction != undefined) {
            const stopButton = document.createElement('button');
            stopButton.textContent = "Back";
            stopButton.classList.add('btn', 'btn-danger');
            stopButton.addEventListener('click', stopAction);
            form.appendChild(stopButton);
        }
        form.appendChild(titleElement);
    }
    addElementsToForm(form, json);
    const buttonGroup = createButtonGroup(json, submitAction, cancelAction);
    if (dialog === "dialog") {
        document.getElementById("modal-footer").innerHTML = '';
        document.getElementById("modal-footer").appendChild(buttonGroup);
        const dialog = bootstrap.Modal.getOrCreateInstance("#extension-dialog");
        dialog.show()
        dialog.on("hidden.bs.modal", () => {
            cancelAction()
        })
    } else {
        form.appendChild(buttonGroup);
    }

}

function addElementsToForm(form, json) {



    const description = document.createElement('p');
    description.textContent = json.description;
    form.appendChild(description);
    if (json.fields) {
        json.fields.forEach(field => {
            const formGroup = createFormGroup(field);
            form.appendChild(formGroup);
        });
    }

    return form;
}

function createTitleElement(json) {
    const title = document.createElement('h1');
    title.textContent = json.title;
    return title;
}

function createFormGroup(field) {
    const formGroup = document.createElement('div');
    formGroup.classList.add('mb-3');

    if (field.label && !field.labelHidden) {
        const label = document.createElement('label');
        label.textContent = field.label;
        label.setAttribute('for', field.key);
        formGroup.appendChild(label);
    }

    const input = createInputElement(field);
    formGroup.appendChild(input);
    return formGroup;
}

function createInputElement(field) {
    let input;

    switch (field.type) {
        case "Console":
            input = document.createElement('pre');
            input.innerHTML = ansi_up.ansi_to_html(field.value || field.placeholder || '');
            input.style.maxHeight = field.lines * 20 + 'px';
            break;
        case "TextArea":
            input = document.createElement('textarea');
            input.rows = field.lines || 3;
            input.textContent = field.value || '';
            break;

        case "Checkbox":
        case "RadioButton":
            input = createCheckboxOrRadioGroup(field);
            break;

        case "Switch":
            input = createSwitchElement(field);
            break;

        case "Select":
            input = document.createElement('select');
            field.items.forEach(item => {
                const option = document.createElement('option');
                option.value = item.value;
                option.text = item.label;
                input.appendChild(option);
            });
            break;

        default:
            input = document.createElement('input');
            input.type = field.type.toLowerCase();
            input.value = field.value;
            break;
    }

    input.id = field.key;
    input.name = field.key;
    if (field.readOnly) input.readOnly = true;
    if (field.type == "Checkbox" || field.type == "RadioButton" || field.type == "Switch") {

    } else {
        if (field.required) input.required = true;
        input.classList.add('form-control');
        if (field.placeholder) input.placeholder = field.placeholder;
    }
    return input;
}

function createCheckboxOrRadioGroup(field) {
    const wrapper = document.createDocumentFragment();

    field.items.forEach(item => {
        const inputWrapper = document.createElement('div');
        inputWrapper.classList.add('form-check');

        const input = document.createElement('input');
        input.type = field.type === "Checkbox" ? 'checkbox' : 'radio';
        input.classList.add('form-check-input');
        input.id = `${field.key}_${item.value}`;
        input.name = field.key; // Grouping by name for radio buttons
        input.value = item.value;
        input.checked = field.value === item.value;

        const itemLabel = document.createElement('label');
        itemLabel.classList.add('form-check-label');
        itemLabel.setAttribute('for', input.id);
        itemLabel.textContent = item.label;

        inputWrapper.appendChild(input);
        inputWrapper.appendChild(itemLabel);
        wrapper.appendChild(inputWrapper);
    });

    return wrapper;
}

function createSwitchElement(field) {
    const switchWrapper = document.createElement('div');
    switchWrapper.classList.add('form-check', 'form-switch');

    const input = document.createElement('input');
    input.type = 'checkbox';
    input.classList.add('form-check-input');
    input.setAttribute('role', 'switch');
    input.id = field.key;
    input.checked = field.value === "true";

    const label = document.createElement('label');
    label.classList.add('form-check-label');
    label.setAttribute('for', field.key);
    label.textContent = field.label;

    switchWrapper.appendChild(input);
    switchWrapper.appendChild(label);

    return switchWrapper;
}

function createButtonGroup(json, submitAction, cancelAction) {
    const buttonGroup = document.createElement('div');
    buttonGroup.classList.add('btn-group');
    json.buttons.forEach(buttonText => {
        const btn = document.createElement('button');
        btn.classList.add('btn',"btn-default");
        buttonGroup.appendChild(btn);
        btn.textContent = buttonText
        if (buttonText=="Cancel") {
            btn.classList.add( 'btn-secondary');
            btn.addEventListener('click', cancelAction);
        }else{
            if (buttonText=="Submit"||buttonText=="Ok")
                btn.classList.add('btn-primary');
            btn.addEventListener('click', submitAction);
        }
        
    })



    return buttonGroup;
}


module.exports = { renderForm };