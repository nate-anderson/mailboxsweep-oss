function handler(event) {
    var request = event.request;


    // if request URI specifies a file extension (like css, js, img)
    // leave the request alone 
    if (/\.\w+$/.test(request.uri)) {
        return request;
    }

    if (request.uri === "/") {
        return request;
    }

    if (!request.uri.endsWith(".html")) {
        if (request.uri.endsWith("/")) {
            request.uri += "index.html"
        } else {
            request.uri += "/index.html"
        }
    }

    return request;
}