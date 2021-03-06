// Copyright 2012 Jason McVetta.  This is Free Software, released under the 
// terms of the GNU Public License version 3.

package ws

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"github.com/jmcvetta/tokenizer"
	"io"
	"log"
)

// Maybe these should be more similar to HTTP response codes.
const (
	invalidRequest = "Invalid Request"
	success        = "Success"
)

type JsonTokenizeRequest struct {
	ReqId string            // Request ID string will be returned unchanged with the response to this request
	Data  map[string]string // Maps fieldnames to text
}

type TokenizeReponse struct {
	ReqId  string            // Request ID string from orginating JsonTokenizeRequest
	Status string            // Status code
	Error  string            // Error message if any
	Data   map[string]string // Maps fieldnames to token strings
}

type DetokenizeRequest struct {
	ReqId string // Request ID string will be returned unchanged with the response to this request
	Data  map[string]string
}

type foundToken struct {
	// Is it really pointful to return the token?
	Token string // The token we looked up
	Found bool   // Was the token found in the database?
	Text  string // The text it represents, if found
}

type DetokenizeReponse struct {
	ReqId  string                // Request ID string from orginating JsonTokenizeRequest
	Status string                // Status code
	Error  string                // Error message if any
	Data   map[string]foundToken // Maps fieldnames to foundToken instances
}

type wsHandler func(ws *websocket.Conn)

func Tokenize(t tokenizer.Tokenizer) wsHandler {
	return func(ws *websocket.Conn) {
		log.Println("New websocket connection")
		log.Println("    Location:  ", ws.Config().Location)
		log.Println("    Origin:    ", ws.Config().Origin)
		log.Println("    Protocol:  ", ws.Config().Protocol)
		dec := json.NewDecoder(ws)
		enc := json.NewEncoder(ws)
		for {
			var err error
			var request JsonTokenizeRequest
			// Read one request from the socket and attempt to decode
			switch err = dec.Decode(&request); true {
			case err == io.EOF:
				log.Println("Websocket disconnecting")
				return
			case err != nil:
				// Request could not be decoded - return error
				response := TokenizeReponse{Status: invalidRequest, Error: err.Error()}
				enc.Encode(&response)
				log.Println("Invalid request - websocket disconnecting")
				return
			}
			data := make(map[string]string)
			for fieldname, text := range request.Data {
				data[fieldname], err = t.Tokenize(text)
				if err != nil {
					// TODO: Do something nicer with this error?
					log.Panic(err)
				}
			}
			response := TokenizeReponse{
				ReqId:  request.ReqId,
				Status: success,
				Data:   data,
			}
			enc.Encode(response)
		}
	}
}

// A websocket handler for detokenization
func Detokenize(t tokenizer.Tokenizer) wsHandler {
	return func(ws *websocket.Conn) {
		dec := json.NewDecoder(ws)
		enc := json.NewEncoder(ws)
		for {
			var request DetokenizeRequest
			// Read one request from the socket and attempt to decode
			switch err := dec.Decode(&request); true {
			case err == io.EOF:
				log.Println("Websocket disconnecting")
				return
			case err != nil:
				// Request could not be decoded - return error
				response := DetokenizeReponse{Status: invalidRequest, Error: err.Error()}
				enc.Encode(&response)
				return
			}
			data := make(map[string]foundToken)
			for fieldname, token := range request.Data {
				ft := foundToken{
					Token: token,
				}
				text, err := t.Detokenize(token)
				switch {
				case nil == err:
					ft.Text = text
					ft.Found = true
				case err == tokenizer.TokenNotFound:
					ft.Found = false
				case err != nil:
					log.Panic(err)
				}
				data[fieldname] = ft
			}
			response := DetokenizeReponse{
				ReqId:  request.ReqId,
				Status: success,
				Data:   data,
			}
			enc.Encode(response)
		}
	}
}
