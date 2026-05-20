package domain

type APIResponse struct {
	OK    bool      `json:"ok"`
	Data  any       `json:"data"`
	Error *APIError `json:"error"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	ErrorRoomExpired         = "ROOM_EXPIRED"
	ErrorRoomNotFound        = "ROOM_NOT_FOUND"
	ErrorParticipantNotFound = "PARTICIPANT_NOT_FOUND"
	ErrorValidation          = "VALIDATION_ERROR"
	ErrorProvider            = "PROVIDER_ERROR"
)

func Success(data any) APIResponse {
	return APIResponse{OK: true, Data: data, Error: nil}
}

func Failure(code string, message string) APIResponse {
	return APIResponse{OK: false, Data: nil, Error: &APIError{Code: code, Message: message}}
}
