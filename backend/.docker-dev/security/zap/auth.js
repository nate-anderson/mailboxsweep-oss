/**
 * Runs in the context of the Graal.js VM in Zap
 */

const HttpRequestHeader = Java.type("org.parosproxy.paros.network.HttpRequestHeader");
const HttpHeader = Java.type("org.parosproxy.paros.network.HttpHeader");
const URI = Java.type("org.apache.commons.httpclient.URI");
const ScriptVars = Java.type('org.zaproxy.zap.extension.script.ScriptVars');
const AuthenticationHelper = Java.type('org.zaproxy.zap.authentication.AuthenticationHelper');

function authenticate(helper, paramsValues, credentials) {
  print("### Using custom authenticator! ###");
  const initLoginEndpoint = new URI(paramsValues.get("loginPostUrl"));;
  const userName = credentials.get("username");

  const initHeader = new HttpRequestHeader(HttpRequestHeader.POST, initLoginEndpoint, HttpHeader.HTTP11);
  const initBody = JSON.stringify({ address: userName });

  const initMsg = helper.prepareMessage();
  initMsg.setRequestHeader(initHeader);
  initMsg.setRequestBody(initBody);
  initMsg.getRequestHeader().setContentLength(msg.getRequestBody().length());

  print(`Going to POST login request to ${paramsValues.get("loginPostUrl")}`);
  helper.sendAndReceive(initMsg, false);
  print("Received init response status code: " + initMsg.getResponseHeader().getStatusCode());
  const initJsonResponse = JSON.parse(initMsg.getResponseBody().toString());
  const initOk = initJsonResponse["ok"];
  if (!initOk) {
    print("Login init not OK!");
  }

  const host = paramsValues.get("host");
  const port = paramsValues.get("tempLoginServerSenderPort");
  const tokenUrl = `http://${host}:${port}?address=${userName}`;
  print(`Going to get login token from URL ${tokenUrl}`);
  const getLoginLinkTempEndpoint = new URI(tokenUrl, false);
  const linkHeader = new HttpRequestHeader(HttpRequestHeader.GET, getLoginLinkTempEndpoint, HttpHeader.HTTP11);
  const linkMsg = helper.prepareMessage();
  linkMsg.setRequestHeader(linkHeader);
  
  print(`Going to GET login token from ${tokenUrl}...`);
  helper.sendAndReceive(linkMsg, false);
  print("Received link response status code: " + linkMsg.getResponseHeader().getStatusCode());

  const tokenJsonResponse = JSON.parse(linkMsg.getResponseBody().toString());
  const link = tokenJsonResponse["link"];
  print("Got login link " + link + ", going to attempt login");

  const loginUri = new URI(link, false);
  const loginMsg = helper.prepareMessage();
  const loginHeader = new HttpRequestHeader(HttpRequestHeader.GET, loginUri, HttpHeader.HTTP11);
  loginMsg.setRequestHeader(loginHeader);

  // SHOULD set X-Mailboxsweep-Session cookie
  helper.sendAndReceive(loginMsg);
}


function getRequiredParamsNames(){
  return ["loginPostUrl", "host", "tempLoginServerSenderPort"];
}


function getOptionalParamsNames(){
  return [];
}


function getCredentialsParamsNames(){
  return ["username"];
}