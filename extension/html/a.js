
import * as a from "./rpc/extension_grpc_web_pb.js";
const client = new ExtensionHostServiceClient('http://localhost:8080');
const request = new GetHelloRequest();
export const getHello = (name) => {
  request.setName(name)
client.getHello(request, {}, (err, response) => {
    console.log(request.getName());
    console.log(response.toObject());
  });
}
getHello("D")