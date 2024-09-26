package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace" // Importar o pacote trace
)

// Variáveis globais
var (
	serviceName = "goGinApp"
)

// Mock do exporter OpenTelemetry
type mockExporter struct{}

func (m *mockExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		log.Printf("Exported span: %v\n", span)
	}
	return nil
}

func (m *mockExporter) Shutdown(ctx context.Context) error {
	log.Println("Mock exporter shutting down")
	return nil
}

func initTracer() func(context.Context) error {
	exporter := &mockExporter{}

	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("library.language", "go"),
		),
	)
	if err != nil {
		log.Printf("Could not set resources: %v", err)
	}

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resources),
		),
	)

	return exporter.Shutdown
}

func main() {
	// Inicializa o tracer
	cleanup := initTracer()
	defer cleanup(context.Background())

	// Configuração do Gin e do Middleware OpenTelemetry
	r := gin.Default()
	r.Use(otelgin.Middleware(serviceName))

	// Define uma rota para a raiz
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to the Go Gin App!",
		})
	})

	// Define rotas simples para teste
	r.GET("/books", func(c *gin.Context) {
		// Adiciona custom attributes
		tracer := otel.Tracer(serviceName)
		_, span := tracer.Start(c.Request.Context(), "BooksHandler")
		defer span.End()

		// Adiciona atributos ao span
		span.SetAttributes(attribute.String("controller", "books"))

		// Adiciona um evento ao span
		span.AddEvent("This is a sample event", trace.WithAttributes(attribute.Int("pid", 4328), attribute.String("sampleAttribute", "Test")))

		c.JSON(http.StatusOK, gin.H{
			"message": "List of books",
		})
	})

	// Roda o servidor Gin (apenas para simulação)
	// Você pode querer remover essa linha se não quiser que o servidor seja realmente iniciado
	go r.Run(":8090")

	// Simular uma requisição para /books para retornar HTTP 200
	res, err := http.Get("http://localhost:8090/books")
	if err != nil {
		log.Fatalf("Error making request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		log.Println("Success! Got HTTP 200 from /books.")
	} else {
		log.Printf("Expected HTTP 200 but got %d", res.StatusCode)
	}
}
