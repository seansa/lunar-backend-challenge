package handler

type acceptedResponse struct {
	Status        string `json:"status"`
	Channel       string `json:"channel"`
	MessageNumber int64  `json:"messageNumber"`
}
