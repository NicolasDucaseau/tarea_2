package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/streadway/amqp"

	ventas "app/tarea_2/pb/pb"

	"google.golang.org/grpc"
)

type server struct {
	ventas.UnimplementedSalesServiceServer
}

func (s *server) PlaceOrder(ctx context.Context, req *ventas.OrderRequest) (*ventas.OrderResponse, error) {

	// Convertir el slice de punteros a slice de valores
	var products []ventas.Product
	for _, p := range req.Products {
		products = append(products, *p)
	}

	// Crear una estructura de datos para los productos (sin información del cliente)
	order := struct {
		Products []ventas.Product
	}{
		Products: products,
	}

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

	// Insertar la orden en la base de datos
	collection := client.Database("aplicacion").Collection("orders")
	insertResult, err := collection.InsertOne(context.TODO(), order)
	if err != nil {
		log.Fatalf("Error al insertar en la base de datos: %v", err)
		return nil, err
	}

	//Para obtener la ID
	objectIDStr := insertResult.InsertedID.(primitive.ObjectID).Hex()

	// Conecta y publica el campo "customer" en RabbitMQ
	err = publishToRabbitMQ(&products, req.Customer, objectIDStr)
	if err != nil {
		log.Printf("Error al publicar en RabbitMQ: %v", err)
	}

	// Devolver la respuesta al cliente con la ID del documento insertado
	return &ventas.OrderResponse{
		Message: fmt.Sprintf("¡La orden se ha procesado con éxito! ID: %v", insertResult.InsertedID),
	}, nil
}

// Función para publicar en RabbitMQ
func publishToRabbitMQ(products *[]ventas.Product, customer *ventas.Customer, Orderid string) error {
	// Conecta a RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return err
	}
	defer conn.Close()

	// Crea un canal
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// Declarar una cola de despacho
	_, err = ch.QueueDeclare(
		"cola_despacho",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Declarar una cola de inventario"
	_, err = ch.QueueDeclare(
		"inventario_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	//declarar una cola de notificaciones
	_, err = ch.QueueDeclare(
		"notificacion_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Convierte la estructura "products" a un formato adecuado para enviarla por RabbitMQ
	productsJSON, err := json.Marshal(products)
	if err != nil {
		log.Printf("Error al convertir productos a JSON: %v\n", err)
		return err
	}

	// Convierte la estructura "customer" a un formato adecuado para enviarla por RabbitMQ
	customerData := []byte(fmt.Sprintf(`{
		"OrderId": "%s",
		"name": "%s",
		"lastname": "%s",
		"email": "%s",
		"location": {
			"address1": "%s",
			"address2": "%s",
			"city": "%s",
			"state": "%s",
			"postalCode": "%s",
			"country": "%s"
		},
		"phone": "%s"
	}`,
		Orderid,
		customer.Name,
		customer.Lastname,
		customer.Email,
		customer.Location.Address1,
		customer.Location.Address2,
		customer.Location.City,
		customer.Location.State,
		customer.Location.PostalCode,
		customer.Location.Country,
		customer.Phone,
	))

	// Publica el mensaje en la cola de RabbitMQ (despacho)
	err = ch.Publish(
		"",
		"cola_despacho",
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        customerData,
		},
	)
	if err != nil {
		return err
	}
	// Publica el mensaje en la cola de RabbitMQ (inventario)
	err = ch.Publish(
		"",
		"inventario_queue",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        productsJSON,
		})
	if err != nil {
		log.Printf("Error al publicar mensaje en RabbitMQ (inventario): %v\n", err)
		return err
	}

	// Combina los dos JSON en una estructura JSON
	notificacionData := []byte(fmt.Sprintf(`{
    	"products": %s,
    	"customer": %s
	}`, productsJSON, customerData))

	// Publica el mensaje en la cola de RabbitMQ (notificacion)
	err = ch.Publish(
		"",
		"notificacion_queue",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        notificacionData,
		})
	if err != nil {
		log.Printf("Error al publicar mensaje en RabbitMQ (inventario): %v\n", err)
		return err
	}
	return nil
}

func main() {
	lis, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatalf("No se pudo escuchar en el puerto: %v", err)
	}
	s := grpc.NewServer()
	ventas.RegisterSalesServiceServer(s, &server{})
	fmt.Println("Servidor gRPC en ejecución...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Error al servir: %v", err)
	}
}
