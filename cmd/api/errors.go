package main

import (
	"fmt"
	"net/http"
)

// logError method is a generic helper for logging an error message. Later this will be upgraded to use
// structured logging, and record additional information about the request including the HTTP method and URL.
func (app *application) logError(r *http.Request, err error) {
	// Use the PrintError method to log the error message, and include the current
	// request method and URL as properties in the log entry
	app.logger.PrintError(err, map[string]string{
		"request_method": r.Method,
		"request_url":    r.URL.String(),
	})
}

// rateLimitExceededResponse is evoked when there's too many request from the client than the server permits
func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(w, r, http.StatusTooManyRequests, "rate limit exceeded")
}

// failedValidationResponse helper writes a 422 Unprocessable Entity and the contents of the errors map from our new
// Validator type as a JSON response body. Note that the errors' parameter here has the type map[string]string, which
// is exactly the same as the errors map contained in our Validator type
func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	app.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

// badRequestResponse for sending a 400 server response code and error back to the client
func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

// errorResponse method is a generic helper for sending JSON-formatted error messages to the client with a given status
// code. Note the use of interface{} type for the message parameter, rather than just a string type, as this gives us
// more flexibility over the values that we can include in the response
func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message interface{}) {
	env := envelope{"error": message}

	// Write the response using the writeJSON() helper. If this happens to return an error then log it, and fall back
	// to sending the client an empty response with a 500 Internal Server Error status code.
	err := app.writeJSON(w, status, env, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(500)
	}
}

// editConflictResponse error will be sent to client, this is used to mitigate a scenario where there's racing condition
// between two Go routine trying to update the same document at the same time
func (app *application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	app.errorResponse(w, r, http.StatusConflict, message)
}

// serverErrorResponse method will be used when our application encounters an unexpected problem at runtime. It logs
// the detailed error message, then uses the errorResponse helper to send a 500 Internal Server Error status code and
// JSON response (containing a generic error message) to the client
func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	message := "the server encountered a problem and could not process your request"
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}

// notFoundResponse method will be used to send a 404 Not Found status code and JSON response to the client.
func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	app.errorResponse(w, r, http.StatusNotFound, message)
}

// methodNotAllowedResponse method will be used to send a 405 Method Not Allowed status code and JSON response to the client
func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}
