package main

type Request struct {
	ID     string `json:"id"`
	TOID   string `json:"toid"`
	Params string `json:"params"`
}

type Response struct {
	ID     string `json:"id"`
	Result string `json:"result"`
}

type Error struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}
