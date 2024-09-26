
const { client,extension } = require('./client.js');
async function listExtensions() {
    $("#extension-list-container").show();
    $("#extension-page-container").hide();

    try {
        const extensionListContainer = document.getElementById('extension-list-container');
        extensionListContainer.innerHTML = ''; // Clear previous entries
        const response = await client.listExtensions(new extension.Empty(), {});
        const header = document.createElement('h1');
        header.classList.add('mb-4');
        header.textContent = "Extension List";
        extensionListContainer.appendChild(header);

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

    listItem.appendChild(contentDiv);

    const switchDiv = createSwitchElement(ext);
    listItem.appendChild(switchDiv);
    const {openExtensionPage} = require('./extensionPage.js');

    listItem.addEventListener('click', () => openExtensionPage(ext.getId()));
    
    return listItem;
}

function createSwitchElement(ext) {
    const switchDiv = document.createElement('div');
    switchDiv.className = 'form-check form-switch';

    const switchButton = document.createElement('input');
    switchButton.type = 'checkbox';
    switchButton.className = 'form-check-input';
    switchButton.checked = ext.getEnable();
    switchButton.addEventListener('change', () => toggleExtension(ext.getId(), switchButton.checked));

    switchDiv.appendChild(switchButton);
    return switchDiv;
}

async function toggleExtension(extensionId, enable) {
    const request = new extension.EditExtensionRequest();
    request.setExtensionId(extensionId);
    request.setEnable(enable);

    try {
        await client.editExtension(request, {});
        console.log(`Extension ${extensionId} updated to ${enable ? 'enabled' : 'disabled'}`);
    } catch (err) {
        console.error('Error updating extension status:', err);
    }
}



module.exports = { listExtensions };