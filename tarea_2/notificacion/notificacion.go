package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"strings"

	"github.com/streadway/amqp"
)

type Notificacion struct {
	OrderID  string               `json:"orderID"`
	GroupID  string               `json:"groupID"`
	Products []Product            `json:"products"`
	Customer CustomerNotificacion `json:"customer"`
}
type Product struct {
	Title       string  `json:"title"`
	Author      string  `json:"author"`
	Genre       string  `json:"genre"`
	Pages       int     `json:"pages"`
	Publication string  `json:"publication"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
}

type Mensaje struct {
	Products []Product `json:"products"`
	Customer Customer  `json:"customer"`
}

type Customer struct {
	OrderID  string   `json:"OrderId"`
	Name     string   `json:"name"`
	Lastname string   `json:"lastname"`
	Email    string   `json:"email"`
	Location Location `json:"location"`
	Phone    string   `json:"phone"`
}

type CustomerNotificacion struct {
	Name     string   `json:"name"`
	Lastname string   `json:"lastname"`
	Email    string   `json:"email"`
	Location Location `json:"location"`
	Phone    string   `json:"phone"`
}

type Location struct {
	Address1   string `json:"address1"`
	Address2   string `json:"address2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

// **************
// POST endpoint
// **************
func post(notificacion Notificacion) {

	//Atributos de request
	url := "https://sjwc0tz9e4.execute-api.us-east-2.amazonaws.com/Prod"
	headers := []string{"Content-Type: application/json"}

	// Convierte el mapa final a JSON
	payload, err := json.Marshal(notificacion)
	if err != nil {
		fmt.Println("Error al convertir mapa final a JSON:", err)
		return
	}

	fmt.Println("Notificacion:", string(payload))

	//Request AWS
	req, err := http.NewRequest("POST", url, strings.NewReader(string(payload)))
	if err != nil {
		log.Fatalf("NewRequest POST error")
	}

	if headers[0] != "" {
		for _, element := range headers {
			header := strings.Split(element, ":")
			req.Header.Add(header[0], header[1])
		}
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Response error")
	}

	defer res.Body.Close()
}

func main() {
	fmt.Println("Servicio de notificaciones en ejecución...")
	// Conexión a RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Error al conectar a RabbitMQ: %v", err)
	}
	defer conn.Close()

	// Crea un canal
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error al abrir un canal: %v", err)
	}
	defer ch.Close()

	// Declarar la misma cola que en ventas.go
	_, err = ch.QueueDeclare(
		"notificacion_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Error al declarar la cola: %v", err)
	}

	// Consumir mensajes de la cola
	msgs, err := ch.Consume(
		"notificacion_queue",
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Error al consumir mensajes: %v", err)
	}

	// Procesar los mensajes recibidos
	for msg := range msgs {
		// El mensaje contiene los datos del cliente en formato JSON
		customerData := msg.Body

		// Decodificar customerData a una estructura mensaje
		var mensaje Mensaje
		err := json.Unmarshal(customerData, &mensaje)
		if err != nil {
			log.Printf("Error al decodificar datos del cliente: %v", err)
			continue
		}

		//crear la notificacion
		var notificacion Notificacion
		notificacion.OrderID = mensaje.Customer.OrderID
		notificacion.GroupID = "6Yt!1C3v#7"
		notificacion.Products = mensaje.Products
		notificacion.Customer.Name = mensaje.Customer.Name
		notificacion.Customer.Lastname = mensaje.Customer.Lastname
		notificacion.Customer.Email = "nicolas.ducaseau.g@gmail.com"
		notificacion.Customer.Location = mensaje.Customer.Location
		notificacion.Customer.Phone = mensaje.Customer.Phone

		//CURL notificacion
		post(notificacion)

		fmt.Println("Mensaje recibido de ventas.go")
	}

	select {}
}
