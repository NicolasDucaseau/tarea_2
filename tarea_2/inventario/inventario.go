package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Product struct {
	Title    string `json:"title"`
	Quantity int    `json:"quantity"`
}

func main() {
	fmt.Println("Servicio de inventario en ejecución...")

	// Configura la conexión a MongoDB
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	// Comprueba si la conexión a MongoDB está activa
	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	// Configura la conexión a RabbitMQ
	amqpURI := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(amqpURI)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	// Declara una cola para recibir mensajes de "ventas"
	queue, err := ch.QueueDeclare(
		"inventario_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Consumir mensajes de la cola
	msgs, err := ch.Consume(
		queue.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Procesa los mensajes de "ventas"
	for msg := range msgs {
		var productsSold []Product

		// Decodifica el mensaje
		if err := json.Unmarshal(msg.Body, &productsSold); err != nil {
			log.Printf("Error al decodificar el mensaje: %v\n", err)
			continue
		}

		// Actualiza el stock en la base de datos según los productos vendidos
		for _, productSold := range productsSold {
			err := updateStock(context.Background(), client, productSold.Title, productSold.Quantity)
			if err != nil {
				log.Printf("Error al actualizar el stock: %v\n", err)
			}
		}
	}
	fmt.Printf("Stock actualizado!")
}

func updateStock(ctx context.Context, client *mongo.Client, title string, quantity int) error {
	collection := client.Database("aplicacion").Collection("products")

	// Busca el producto por título
	filter := bson.M{"title": title}
	update := bson.M{
		"$inc": bson.M{"quantity": -quantity},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}
