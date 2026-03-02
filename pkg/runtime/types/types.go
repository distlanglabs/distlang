package types

// Request represents a minimal HTTP-like request.
type Request struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    string
}

// Response represents a minimal HTTP-like response.
type Response struct {
	Status  int
	Headers map[string]string
	Body    string
}
