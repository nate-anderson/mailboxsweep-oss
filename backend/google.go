package mailboxsweep

// OAuth2 scopes for the application to work fully
var FullGoogleAPIScopes = []string{
	"https://mail.google.com/",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
}

// Bare minimum OAuth2 scopes, "mail.google.com" will need to be requested later
var BasicGoogleAPIScopes = []string{
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
}
