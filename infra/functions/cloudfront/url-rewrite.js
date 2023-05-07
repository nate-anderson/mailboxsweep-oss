function handler(event) {
    var request = event.request;
    if (!request.uri.endsWith("index.html")) {
        request.uri += "/index.html"
    }
    return request;
}