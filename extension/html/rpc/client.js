const extension = require("./extension_grpc_web_pb.js");

const grpcServerAddress = '/';
const client = new extension.ExtensionHostServicePromiseClient(grpcServerAddress, null, null);

module.exports = { client ,extension};