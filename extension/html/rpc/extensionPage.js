const { extensionClient } = require('./client.js');
const extension = require("./extension_grpc_web_pb.js");

const { renderForm } = require('./formRenderer.js');
const { listExtensions } = require('./extensionList.js');
var currentExtensionId = undefined;
function openExtensionPage(extensionId) {
    currentExtensionId = extensionId;
    $("#extension-list-container").hide();
    $("#extension-page-container").show();
    $("#connection-page").hide();
    connect()
}

function connect() {
    const request = new extension.ExtensionRequest();
    request.setExtensionId(currentExtensionId);

    const stream = extensionClient.connect(request, {});

    stream.on('data', (response) => {
        console.log('Receving ', response);
        if (response.getExtensionId() === currentExtensionId) {
            ui = JSON.parse(response.getJsonUi())
            if (response.getType() == proto.hiddifyrpc.ExtensionResponseType.SHOW_DIALOG) {
                renderForm(ui, "dialog", handleSubmitButtonClick, undefined);
            } else {
                renderForm(ui, "", handleSubmitButtonClick, handleStopButtonClick);
            }


        }
    });

    stream.on('error', (err) => {
        console.error('Error opening extension page:', err);
        // openExtensionPage(extensionId);
    });

    stream.on('end', () => {
        console.log('Stream ended');
        setTimeout(connect, 1000);

    });
}

async function handleSubmitButtonClick(event, button) {
    event.preventDefault();
    bootstrap.Modal.getOrCreateInstance("#extension-dialog").hide();
    const request = new extension.SendExtensionDataRequest();
    request.setButton(button);
    if (event.type != 'hidden.bs.modal') {
        const formData = new FormData(event.target.closest('form'));
        const datamap = request.getDataMap()
        formData.forEach((value, key) => {
            datamap.set(key, value);
        });
    }
    request.setExtensionId(currentExtensionId);

    try {
        await extensionClient.submitForm(request, {});
        console.log('Form submitted successfully.');
    } catch (err) {
        console.error('Error submitting form:', err);
    }
}


async function handleStopButtonClick(event) {
    event.preventDefault();
    const request = new extension.ExtensionRequest();
    request.setExtensionId(currentExtensionId);
    bootstrap.Modal.getOrCreateInstance("#extension-dialog").hide();
    try {
        await extensionClient.stop(request, {});
        console.log('Extension stopped successfully.');
        currentExtensionId = undefined;
        listExtensions(); // Return to the extension list
    } catch (err) {
        console.error('Error stopping extension:', err);
    }
}



module.exports = { openExtensionPage };