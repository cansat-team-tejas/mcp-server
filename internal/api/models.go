package api

type AskRequest struct {
	Question string `json:"question"`
}

type QueryRequest struct {
	SQL string `json:"sql"`
}
