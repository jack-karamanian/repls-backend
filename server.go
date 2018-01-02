package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn      *websocket.Conn
	mt        int
	container *Container
}

type ImageEntry struct {
	ImageName   string `json:"imageName"`
	DisplayName string `json:"displayName"`
}

type ServerError struct {
	Error string `json:"error"`
}

var addr = flag.String("addr", "127.0.0.1:8080", "http address")

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (client *Client) HandleOutput() {
	for {

		for output := range client.container.Stdout {

			err := client.conn.WriteMessage(client.mt, []byte(output))
			if err != nil {
				log.Println(err)
				return
			}

		}

	}

}

func (client *Client) HandleInput() {
	for {
		_, message, err := client.conn.ReadMessage()

		if err != nil {
			log.Println(err)
			return
		}

		fmt.Println(string(message))
		client.container.Stdin <- string(message)
	}
}

func ws(writer http.ResponseWriter, req *http.Request) {
	images := req.URL.Query()["image"]

	if len(images) != 1 {
		encoder := json.NewEncoder(writer)
		serverError := ServerError{
			Error: "1 image must be specified",
		}
		encoder.Encode(&serverError)
		req.Response.StatusCode = http.StatusBadRequest

		return
	}

	image := images[0]

	fmt.Println(image)

	sock, err := upgrader.Upgrade(writer, req, nil)

	if err != nil {
		panic(err)
	}

	client := &Client{
		conn:      sock,
		mt:        websocket.TextMessage,
		container: &Container{},
	}

	closeHandler := client.conn.CloseHandler()

	client.conn.SetCloseHandler(func(code int, text string) error {
		go client.container.Stop()
		return closeHandler(code, text)
	})

	client.container.Start(image)

	go client.HandleInput()
	go client.HandleOutput()
}

func GetImages(writer http.ResponseWriter, req *http.Request) {

	data, err := ioutil.ReadFile("./images.json")

	if err != nil {
		fmt.Print(err)
		writer.Write([]byte(fmt.Sprint(err)))
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(data)
}

func CrossOrigin(orig func(http.ResponseWriter, *http.Request)) func(writer http.ResponseWriter, req *http.Request) {
	return func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Access-Control-Allow-Origin", req.Header.Get("Origin"))
		orig(writer, req)
	}

}

func Serve() {
	http.HandleFunc("/ws", ws)
	http.HandleFunc("/images", CrossOrigin(GetImages))
	http.ListenAndServe(*addr, nil)
}
