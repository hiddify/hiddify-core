
const { extensionClient } = require('./client.js');
const extension = require("./extension_grpc_web_pb.js");
async function listExtensions() {
    $("#extension-list-container").show();
    $("#extension-page-container").hide();
    $("#connection-page").show();

    try {
        const extensionListContainer = document.getElementById('extension-list');
        extensionListContainer.innerHTML = ''; // Clear previous entries
        const response = await extensionClient.listExtensions(new extension.Empty(), {});
        
        const extensionList = response.getExtensionsList();
        extensionList.forEach(ext => {
            const listItem = createExtensionListItem(ext);
            extensionListContainer.appendChild(listItem);
        });
    } catch (err) {
        console.error('Error listing extensions:', err);
    }
}

function createExtensionListItem(ext) {
    const listItem = document.createElement('li');
    listItem.className = 'list-group-item d-flex justify-content-between align-items-center';
    listItem.setAttribute('data-extension-id', ext.getId());

    const contentDiv = document.createElement('div');

    const titleElement = document.createElement('span');
    titleElement.innerHTML = `<strong>${ext.getTitle()}</strong>`;
    contentDiv.appendChild(titleElement);

    const descriptionElement = document.createElement('p');
    descriptionElement.className = 'mb-0';
    descriptionElement.textContent = ext.getDescription();
    contentDiv.appendChild(descriptionElement);
    contentDiv.style.width="100%";
    listItem.appendChild(contentDiv);

    const switchDiv = createSwitchElement(ext);
    listItem.appendChild(switchDiv);
    const {openExtensionPage} = require('./extensionPage.js');

    contentDiv.addEventListener('click', () =>{ 
        if (!ext.getEnable() ){
            alert("Extension is not enabled")
            return
        }
        openExtensionPage(ext.getId())
    });
    
    return listItem;
}

function createSwitchElement(ext) {
    const switchDiv = document.createElement('div');
    switchDiv.className = 'form-check form-switch';

    const switchButton = document.createElement('input');
    switchButton.type = 'checkbox';
    switchButton.className = 'form-check-input';
    switchButton.checked = ext.getEnable();
    switchButton.addEventListener('change', (e) => {
        
        toggleExtension(ext.getId(), switchButton.checked)
    });

    switchDiv.appendChild(switchButton);
    return switchDiv;
}

async function toggleExtension(extensionId, enable) {
    const request = new extension.EditExtensionRequest();
    request.setExtensionId(extensionId);
    request.setEnable(enable);

    try {
        await extensionClient.editExtension(request, {});
        console.log(`Extension ${extensionId} updated to ${enable ? 'enabled' : 'disabled'}`);
    } catch (err) {
        console.error('Error updating extension status:', err);
    }
    listExtensions();
}



module.exports = { listExtensions };