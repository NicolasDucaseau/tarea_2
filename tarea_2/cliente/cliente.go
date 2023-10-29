package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	ventas "app/tarea_2/pb/pb"
)

func main() {
	// Leer el archivo JSON
	jsonData, err := os.ReadFile("products.json")
	if err != nil {
		log.Fatalf("Error al leer el archivo JSON: %v", err)
	}

	// Analizar el archivo JSON en una estructura de datos
	var orderRequest ventas.OrderRequest
	err = json.Unmarshal(jsonData, &orderRequest)
	if err != nil {
		log.Fatalf("Error al analizar el archivo JSON: %v", err)
	}

	// Establecer conexión con el servidor gRPC de ventas
	conn, err := grpc.Dial("localhost:8000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("No se pudo conectar al servidor: %v", err)
	}
	defer conn.Close()

	// Crear un cliente gRPC
	client := ventas.NewSalesServiceClient(conn)

	// Llamar al servicio PlaceOrder
	response, err := client.PlaceOrder(context.Background(), &orderRequest)
	if err != nil {
		log.Fatalf("Error al llamar al servicio PlaceOrder: %v", err)
	}

	// Manejar la respuesta del servidor
	fmt.Printf("Mensaje de confirmación: %s\n", response.Message)

}
