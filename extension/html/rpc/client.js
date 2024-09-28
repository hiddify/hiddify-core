const hiddify = require("./hiddify_grpc_web_pb.js");
const extension = require("./extension_grpc_web_pb.js");

const grpcServerAddress = '/';
const extensionClient = new extension.ExtensionHostServicePromiseClient(grpcServerAddress, null, null);
const hiddifyClient = new hiddify.CorePromiseClient(grpcServerAddress, null, null);

module.exports = { extensionClient ,hiddifyClient};