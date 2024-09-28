const { listExtensions } = require('./extensionList.js');
const { openConnectionPage } = require('./connectionPage.js');
window.onload = () => {
    listExtensions();
    openConnectionPage();
};


