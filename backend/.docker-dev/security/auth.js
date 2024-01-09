/**
 * Runs in the context of the Graal.js VM in Zap
 */

const HttpRequestHeader = Java.type("org.parosproxy.paros.network.HttpRequestHeader");
const HttpHeader = Java.type("org.parosproxy.paros.network.HttpHeader");
const URI = Java.type("org.apache.commons.httpclient.URI");
const ScriptVars = Java.type('org.zaproxy.zap.extension.script.ScriptVars');
const AuthenticationHelper = Java.type('org.zaproxy.zap.authentication.AuthenticationHelper');
const LoggerManager = Java.type("org.apache.logging.log4j.LogManager");
const log = LoggerManager.getLogger("script");

function authenticate(helper, paramsValues, credentials) {
  const initLoginEndpoint = new URI(paramsValues.get("loginPostUrl"), false);
  const userName = credentials.getParam("username");

  const initHeader = new HttpRequestHeader(HttpRequestHeader.POST, initLoginEndpoint, HttpHeader.HTTP11);
  initHeader.setHeader(HttpHeader.CONTENT_TYPE,"application/json");
  const initBody = JSON.stringify({ address: userName });
  
  const initMsg = helper.prepareMessage();
  initMsg.setRequestHeader(initHeader);
  initMsg.setRequestBody(initBody);
  initMsg.getRequestHeader().setContentLength(initMsg.getRequestBody().length());

  helper.sendAndReceive(initMsg, false);
  AuthenticationHelper.addAuthMessageToHistory(initMsg);

  const initJsonResponse = JSON.parse(initMsg.getResponseBody().toString());
  const initOk = initJsonResponse["ok"];
  if (!initOk) {
    log.error("Login init not OK!");
  }

  const host = paramsValues.get("host");
  const port = paramsValues.get("tempLoginServerSenderPort");
  const tokenUrl = `http://${host}:${port}/?address=${encodeURIComponent(userName)}`;
  const getLoginLinkTempEndpoint = new URI(tokenUrl, true);
  const linkHeader = new HttpRequestHeader(HttpRequestHeader.GET, getLoginLinkTempEndpoint, HttpHeader.HTTP11);

  const linkMsg = helper.prepareMessage();
  linkMsg.setRequestHeader(linkHeader);

  helper.sendAndReceive(linkMsg, false);
  AuthenticationHelper.addAuthMessageToHistory(linkMsg);
  print("Received link response status code: " + linkMsg.getResponseHeader().getStatusCode());

  const tokenJsonResponse = JSON.parse(linkMsg.getResponseBody().toString());
  const token = tokenJsonResponse["token"];
  const signature = tokenJsonResponse["signature"];
  const link = `${paramsValues.get("loginPostUrl")}/${token}/${signature}`;

  const loginUri = new URI(link, false);
  const loginMsg = helper.prepareMessage();
  const loginHeader = new HttpRequestHeader(HttpRequestHeader.GET, loginUri, HttpHeader.HTTP11);
  loginMsg.setRequestHeader(loginHeader);

  // SHOULD set X-Mailboxsweep-Session cookie
  helper.sendAndReceive(loginMsg, true);
  print(loginMsg.getResponseHeader());

  return loginMsg;
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