package main

type Request struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params string `json:"params"`
}

type Response struct {
	ID     int    `json:"id"`
	Result string `json:"result"`
}

type Error struct {
	ID    int    `json:"id"`
	Error string `json:"error"`
}
