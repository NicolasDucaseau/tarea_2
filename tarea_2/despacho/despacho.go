package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/streadway/amqp" // Importa la biblioteca de RabbitMQ
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Order struct {
	OrderId string `json:"OrderId"`
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

type Customer struct {
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

type Delivery struct {
	ShippingAddress Location `json:"shippingAddress"`
	ShippingMethod  string   `json:"shippingMethod"`
	TrackingNumber  string   `json:"trackingNumber"`
}

func main() {
	fmt.Println("Servicio de despacho en ejecución...")

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

	var uri = "mongodb://localhost:27017"

	// Conexión a MongoDB
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	// Seleccionar la base de datos y la colección
	collection := client.Database("aplicacion").Collection("orders")

	// Declarar la misma cola que en ventas.go
	_, err = ch.QueueDeclare(
		"cola_despacho",
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
		"cola_despacho",
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

		// Decodificar customerData a una estructura Customer
		var newCustomer Customer
		var OrderId Order
		err := json.Unmarshal(customerData, &newCustomer)
		if err != nil {
			log.Printf("Error al decodificar datos del cliente: %v", err)
			continue
		}
		err = json.Unmarshal(customerData, &OrderId)
		if err != nil {
			log.Printf("Error al decodificar datos del cliente: %v", err)
			continue
		}
		log.Printf("order: %s", OrderId.OrderId)
		log.Printf("customer: %s", newCustomer.Name)

		//Se crea la estructura de deliveries
		var newDelivery Delivery
		newDelivery.ShippingAddress = newCustomer.Location
		newDelivery.ShippingMethod = "USPS"
		newDelivery.TrackingNumber = "12345678901234567890"

		// Convierte la cadena en un ObjectID
		orderID, err := primitive.ObjectIDFromHex(OrderId.OrderId)
		if err != nil {
			log.Printf("Error al decodificar ID: %v", err)
		}

		// Realizar la actualización en la base de datos MongoDB
		filter := bson.M{"_id": orderID}
		update := bson.M{
			"$set": bson.M{
				"customer":   newCustomer,
				"deliveries": []Delivery{newDelivery},
			},
		}
		_, err = collection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			log.Printf("Error al actualizar la base de datos: %v", err)
		}

		fmt.Printf("Mensaje recibido de ventas.go")
	}

	select {}
}
