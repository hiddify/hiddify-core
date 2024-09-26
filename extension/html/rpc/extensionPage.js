const { client,extension } = require('./client.js');
const { renderForm } = require('./formRenderer.js');
const { listExtensions } = require('./extensionList.js');
var currentExtensionId=undefined;
function openExtensionPage(extensionId) {
    currentExtensionId=extensionId;
        $("#extension-list-container").hide();
    $("#extension-page-container").show();
    const request = new extension.ExtensionRequest();
    request.setExtensionId(extensionId);

    const stream = client.connect(request, {});
    
    stream.on('data', (response) => {
        
        if (response.getExtensionId() === currentExtensionId) {
            ui=JSON.parse(response.getJsonUi())
            if(response.getType()== proto.hiddifyrpc.ExtensionResponseType.SHOW_DIALOG) {
                renderForm(ui, "dialog",handleSubmitButtonClick,handleCancelButtonClick,undefined);
            }else{
                renderForm(ui, "",handleSubmitButtonClick,handleCancelButtonClick,handleStopButtonClick);
            }

            
        }
    });
    
    stream.on('error', (err) => {
        console.error('Error opening extension page:', err);
    });
    
    stream.on('end', () => {
        console.log('Stream ended');
    });
}

async function handleSubmitButtonClick(event) {
    event.preventDefault();
    const formData = new FormData(event.target.closest('form'));
    const request = new extension.ExtensionRequest();

    formData.forEach((value, key) => {
        request.getDataMap()[key] = value;
    });
    request.setExtensionId(currentExtensionId);

    try {
        await client.submitForm(request, {});
        console.log('Form submitted successfully.');
    } catch (err) {
        console.error('Error submitting form:', err);
    }
}

async function handleCancelButtonClick(event) {
    event.preventDefault();
    const request = new extension.ExtensionRequest();
    request.setExtensionId(currentExtensionId);

    try {
        await client.cancel(request, {});
        console.log('Extension cancelled successfully.');
    } catch (err) {
        console.error('Error cancelling extension:', err);
    }
}

async function handleStopButtonClick(event) {
    event.preventDefault();
    const request = new extension.ExtensionRequest();
    request.setExtensionId(currentExtensionId);

    try {
        await client.stop(request, {});
        console.log('Extension stopped successfully.');
        currentExtensionId = undefined;
        listExtensions(); // Return to the extension list
    } catch (err) {
        console.error('Error stopping extension:', err);
    }
}



module.exports = { openExtensionPage };