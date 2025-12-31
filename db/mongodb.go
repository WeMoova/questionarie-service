package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoDB wraps the MongoDB client and database
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// NewMongoDB creates a new MongoDB connection
func NewMongoDB() (*MongoDB, error) {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		return nil, fmt.Errorf("MONGODB_URI environment variable is not set")
	}

	dbName := os.Getenv("MONGODB_DATABASE")
	if dbName == "" {
		dbName = "questionarie_db" // default database name
	}

	timeoutStr := os.Getenv("MONGODB_TIMEOUT")
	timeout := 10 * time.Second
	if timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Set client options
	clientOptions := options.Client().
		ApplyURI(mongoURI).
		SetServerSelectionTimeout(timeout).
		SetConnectTimeout(timeout).
		SetMaxPoolSize(100).
		SetMinPoolSize(10)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	log.Printf("Successfully connected to MongoDB database: %s", dbName)

	return &MongoDB{
		Client:   client,
		Database: client.Database(dbName),
	}, nil
}

// Close closes the MongoDB connection
func (m *MongoDB) Close(ctx context.Context) error {
	if m.Client != nil {
		return m.Client.Disconnect(ctx)
	}
	return nil
}

// HealthCheck checks if the MongoDB connection is alive
func (m *MongoDB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return m.Client.Ping(ctx, readpref.Primary())
}

// Collection is a helper method to get a collection from the database
func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

// Collections returns all collection names
func (m *MongoDB) Collections() []string {
	return []string{
		"companies",
		"questionnaires",
		"company_questionnaires",
		"user_questionnaire_assignments",
		"users_metadata",
	}
}
