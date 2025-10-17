package api

type AskRequest struct {
	Question string `json:"question"`
	Filename string `json:"filename"`
}

type QueryRequest struct {
	SQL string `json:"sql"`
}

type DataRequest struct {
	Filename string `json:"filename"`
}
