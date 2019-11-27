package httpbin

type UrlItem struct {
	Method string
	Name   string
	Url    string
	Desc   string
}

var URL_GROUP_CONFIG = []string{
	"This Page",
	"HTTP Methods",
	"Auth",
	"Status Code",
	"Images",
	"Request inspection",
	"Dynamic data",
}

var URL_CONFIG = map[string][]UrlItem{
	"This Page": {
		UrlItem{"GET", "/", "/", ""},
	},
	"HTTP Methods": {
		UrlItem{"GET", "- /delete", "#", "Returns GET data."},
		UrlItem{"GET", "/get", "/get", "Returns GET data."},
		UrlItem{"GET", "- /patch", "#", "Returns GET data."},
		UrlItem{"GET", "- /post", "#", "Returns GET data."},
		UrlItem{"GET", "- /put", "#", "Returns GET data."},
	},
	"Auth": {
		UrlItem{"GET", "/basic-auth/{user}/{passwd}", "/basic-auth/user/passwd", "Challenges HTTPBasic Auth"},
		UrlItem{"GET", "- /bearer", "#", "Challenges HTTPBasic Auth"},
		UrlItem{"GET", "- /digest-auth/{qop}/{user}/{passwd}", "#", "Challenges HTTPBasic Auth"},
		UrlItem{"GET", "- /digest-auth/{qop}/{user}/{passwd}/{algorithm}", "#", "Challenges HTTPBasic Auth"},
		UrlItem{"GET", "- /hidden-basic-auth/{user}/{passwd}", "#", "Challenges HTTPBasic Auth"},
	},
	"Status Code": {
		UrlItem{"GET", "- /status/{code}", "#", ""},
	},
	"Images": {
		UrlItem{"GET", "/image", "/image", "Returns page containing an image based on sent Accept header."},
		UrlItem{"GET", "/image/png", "/image/png", "Returns a PNG image."},
		UrlItem{"GET", "/image/jpeg", "/image/jpeg", "Returns a JPEG image."},
		UrlItem{"GET", "/image/webp", "/image/webp", "Returns a WEBP image."},
		UrlItem{"GET", "/image/svg", "/image/svg", "Returns a SVG image."},
	},
	"Request inspection": {
		UrlItem{"GET", "/ip", "/ip", "Returns Origin IP."},
		UrlItem{"GET", "/user-agent", "/user-agent", "Returns user-agent."},
		UrlItem{"GET", "/headers", "/headers", "Returns header dict."},
	},
	"Dynamic data": {
		UrlItem{"GET", "/base64/{value}", "/base64/aGVsbG8gd29ybGQNCg==", "Decodes base64url-encoded string."},
		UrlItem{"GET", "/bytes/{n}", "/bytes/1024", "Generates <em>n</em> random bytes of binary data, accepts optional <em>seed</em> integer parameter."},
		UrlItem{"GET", "/stream-bytes/{n}", "/stream-bytes/20925?filename=data.bin", "Streams <em>n</em> random bytes of binary data in chunked encoding, accepts optional <em>seed</em>, <em>filename</em> and <em>chunk_size</em> integer parameters."},
		UrlItem{"GET", "/uuid", "/uuid", "Returns UUID4."},
	},
}
