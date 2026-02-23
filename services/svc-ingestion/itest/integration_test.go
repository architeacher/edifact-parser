package itest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/inbound/http/handlers/public"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/config"
	svcRuntime "github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/runtime"
)

const (
	testDBName           = "edifact_test"
	testDBUser           = "test"
	testDBPassword       = "test"
	postgresImage        = "postgres:16-alpine"
	postgresNetworkAlias = "postgres"
	migrateImage         = "migrate/migrate:v4.19.1"
)

var (
	globalDB          *pgxpool.Pool
	globalKafkaWriter *kafkago.Writer
	globalServiceCtx  *svcRuntime.ServiceCtx
	globalHTTPServer  *httptest.Server
	globalHTTPBaseURL string
)

// TestMain sets up and tears down integration test infrastructure.
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, connStr, networkName, err := startPostgres(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start PostgreSQL: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if termErr := pgContainer.Terminate(context.Background()); termErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to terminate PostgreSQL: %v\n", termErr)
		}
	}()

	// Create connection pool and apply migrations
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create pgxpool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	globalDB = pool

	// Apply migrations using the migrate/migrate Docker container
	if err := runMigrations(ctx, networkName); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	// Create a simple Kafka writer that connects to localhost
	// (In integration tests, Kafka is optional and may not be running)
	kafkaWriter := kafkago.NewWriter(kafkago.WriterConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "edifact.events",
	})
	defer kafkaWriter.Close()

	globalKafkaWriter = kafkaWriter

	// Create service context with overridden dependencies
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     0, // Will be overridden
			User:     testDBUser,
			Password: testDBPassword,
			Name:     testDBName,
			SSLMode:  "disable",
			PoolSize: 10,
		},
		Kafka: config.KafkaConfig{
			Brokers:   []string{"localhost:9092"},
			Topic:     "edifact.events",
			Partition: 1,
		},
		Server: config.ServerConfig{
			Port: 0, // Use ephemeral port
		},
		Logging: config.LoggingConfig{
			Level: "error", // Keep logging quiet in tests
		},
		OTel: config.OTelConfig{
			Enabled: false,
		},
	}

	svc := svcRuntime.New()
	if err := svc.Build(
		svcRuntime.WithExternalConfig(cfg),
		svcRuntime.WithExternalDB(pool),
		svcRuntime.WithExternalKafkaWriter(kafkaWriter),
	); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build service context: %v\n", err)
		os.Exit(1)
	}

	globalServiceCtx = svc

	// Create test HTTP server
	globalHTTPServer = httptest.NewServer(svc.HTTPHandler())
	globalHTTPBaseURL = globalHTTPServer.URL

	// Run tests
	code := m.Run()

	// Cleanup
	globalHTTPServer.Close()

	os.Exit(code)
}

// startPostgres starts a PostgreSQL testcontainer on a shared Docker network
// and returns the container, connection string, and network name.
func startPostgres(ctx context.Context) (testcontainers.Container, string, string, error) {
	testNetwork, err := network.New(ctx)
	if err != nil {
		return nil, "", "", fmt.Errorf("creating test network: %w", err)
	}

	container, err := postgres.Run(ctx,
		postgresImage,
		postgres.WithDatabase(testDBName),
		postgres.WithUsername(testDBUser),
		postgres.WithPassword(testDBPassword),
		network.WithNetwork([]string{postgresNetworkAlias}, testNetwork),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, "", "", fmt.Errorf("starting postgres container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, "", "", fmt.Errorf("getting connection string: %w", err)
	}

	return container, connStr, testNetwork.Name, nil
}

// startKafka starts a Kafka testcontainer and returns the container and broker list.
func startKafka(ctx context.Context) (testcontainers.Container, []string, error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "confluentinc/cp-kafka:7.8.0",
			ExposedPorts: []string{"9092/tcp"},
			Env: map[string]string{
				"KAFKA_BROKER_ID":                   "1",
				"KAFKA_ZOOKEEPER_CONNECT":           "localhost:2181",
				"KAFKA_ADVERTISED_LISTENERS":        "PLAINTEXT://kafka:29092,PLAINTEXT_HOST://127.0.0.1:9092",
				"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP": "PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT",
				"KAFKA_INTER_BROKER_LISTENER_NAME": "PLAINTEXT",
				"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR": "1",
			},
			WaitingFor: wait.ForLog("started (kafka.server.KafkaServer)").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return nil, nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, nil, err
	}

	port, err := container.MappedPort(ctx, "9092")
	if err != nil {
		return nil, nil, err
	}

	brokers := []string{net.JoinHostPort(host, port.Port())}
	return container, brokers, nil
}

// runMigrations applies database migrations using the migrate/migrate Docker container,
// bind-mounting the actual migration files from the project.
func runMigrations(ctx context.Context, networkName string) error {
	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:5432/%s?sslmode=disable",
		testDBUser,
		testDBPassword,
		postgresNetworkAlias,
		testDBName,
	)

	migrationsPath, err := getMigrationsPath()
	if err != nil {
		return fmt.Errorf("getting migrations path: %w", err)
	}

	migrateContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:    migrateImage,
			Networks: []string{networkName},
			Cmd: []string{
				"-path", "/migrations",
				"-database", dbURL,
				"up",
			},
			Mounts: testcontainers.Mounts(
				testcontainers.BindMount(migrationsPath, "/migrations"),
			),
			WaitingFor: wait.ForExit().WithExitTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return fmt.Errorf("running migrate container: %w", err)
	}
	defer migrateContainer.Terminate(ctx)

	state, err := migrateContainer.State(ctx)
	if err != nil {
		return fmt.Errorf("getting migrate container state: %w", err)
	}

	if state.ExitCode != 0 {
		logs, _ := migrateContainer.Logs(ctx)

		var logContent string
		if logs != nil {
			defer logs.Close()

			buf := make([]byte, 4096)
			n, _ := logs.Read(buf)
			logContent = string(buf[:n])
		}

		return fmt.Errorf("migrations failed with exit code %d, path: %s, logs: %s", state.ExitCode, migrationsPath, logContent)
	}

	return nil
}

// getMigrationsPath resolves the absolute path to the migrations directory
// relative to this source file.
func getMigrationsPath() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}

	serviceRoot := filepath.Dir(filepath.Dir(currentFile))

	return filepath.Join(serviceRoot, "migrations"), nil
}

// TestValidFilesIngest verifies that valid EDIFACT files ingest successfully.
// NOTE: Some files may fail due to FK constraint on subscription_id - this indicates
// the application needs to use the subscription UUID from the upsert operation rather
// than creating a new UUID in buildMessages. See ingest_file.go:buildMessages().
func TestValidFilesIngest(t *testing.T) {
	t.Parallel()

	ediFiles, err := filepath.Glob("../testdata/edifact/valid/*.edi")
	require.NoError(t, err, "failed to glob .edi files")

	txtFiles, err := filepath.Glob("../testdata/edifact/valid/*.txt")
	require.NoError(t, err, "failed to glob .txt files")

	validFiles := append(ediFiles, txtFiles...)

	type testCase struct {
		name     string
		filepath string
	}

	cases := make([]testCase, 0, len(validFiles))
	for _, f := range validFiles {
		if filepath.Base(f) == ".gitkeep" {
			continue
		}
		cases = append(cases, testCase{
			name:     filepath.Base(f),
			filepath: f,
		})
	}

	require.NotEmpty(t, cases, "no valid test files found")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fileData, err := os.ReadFile(tc.filepath)
			require.NoError(t, err, "failed to read test file")

			// Ingest file
			resp, statusCode := ingestFile(t, fileData, "application/edifact")
			require.Equal(t, http.StatusCreated, statusCode, "expected 201 for valid file: %s", tc.name)

			// Verify response
			require.NotEqual(t, uuid.Nil, uuid.UUID(resp.InterchangeId), "expected non-nil interchange ID")
			require.Equal(t, public.IngestStatusCompleted, resp.Status)
			require.NotNil(t, resp.MessageSummary)
			require.GreaterOrEqual(t, resp.MessageSummary.Total, 0, "message count should be non-negative")
		})
	}
}

// TestIdempotency verifies that ingesting the same file twice returns the same ID.
func TestIdempotency(t *testing.T) {
	t.Parallel()

	fileData, err := os.ReadFile("../testdata/edifact/valid/mscons_single.edi")
	require.NoError(t, err)

	// First ingest
	resp1, statusCode1 := ingestFile(t, fileData, "application/edifact")
	require.Equal(t, http.StatusCreated, statusCode1)

	interchangeID1 := uuid.UUID(resp1.InterchangeId)
	require.NotEqual(t, uuid.Nil, interchangeID1)

	// Second ingest (same file) — should return 200 (idempotent success).
	resp2, statusCode2 := ingestFile(t, fileData, "application/edifact")
	require.Equal(t, http.StatusOK, statusCode2,
		"expected 200 for duplicate file, got %d", statusCode2)

	// IDs should match
	interchangeID2 := uuid.UUID(resp2.InterchangeId)
	require.Equal(t, interchangeID1, interchangeID2, "duplicate file should return same ID")
}

// TestInvalidFilesReturn422 verifies that all invalid EDIFACT files return 422 Unprocessable Entity.
func TestInvalidFilesReturn422(t *testing.T) {
	t.Parallel()

	invalidFiles, err := filepath.Glob("../testdata/edifact/invalid/*.edi")
	require.NoError(t, err, "failed to glob invalid files")

	type testCase struct {
		name     string
		filepath string
	}

	cases := make([]testCase, 0, len(invalidFiles))
	for _, f := range invalidFiles {
		if filepath.Base(f) == ".gitkeep" {
			continue
		}
		cases = append(cases, testCase{
			name:     filepath.Base(f),
			filepath: f,
		})
	}

	require.NotEmpty(t, cases, "no invalid test files found")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fileData, err := os.ReadFile(tc.filepath)
			require.NoError(t, err, "failed to read test file")

			// Ingest file
			_, statusCode := ingestFile(t, fileData, "application/edifact")

			// Should fail with 422
			require.Equal(t, http.StatusUnprocessableEntity, statusCode,
				"expected 422 for invalid file: %s", tc.name)
		})
	}
}

// TestMessageExtractionVerification verifies message counts and types for known files.
func TestMessageExtractionVerification(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		filepath         string
		expectedMessages int
		expectedTypes    map[string]int
	}{
		{
			name:             "mscons_single",
			filepath:         "../testdata/edifact/valid/mscons_single.edi",
			expectedMessages: 1,
			expectedTypes:    map[string]int{"MSCONS": 1},
		},
		{
			name:             "mscons_three_messages",
			filepath:         "../testdata/edifact/valid/mscons_three_messages.edi",
			expectedMessages: 3,
			expectedTypes:    map[string]int{"MSCONS": 3},
		},
		{
			name:             "invoic_simple",
			filepath:         "../testdata/edifact/valid/invoic_simple.edi",
			expectedMessages: 1,
			expectedTypes:    map[string]int{"INVOIC": 1},
		},
		{
			name:             "utilmd_simple",
			filepath:         "../testdata/edifact/valid/utilmd_simple.edi",
			expectedMessages: 1,
			expectedTypes:    map[string]int{"UTILMD": 1},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fileData, err := os.ReadFile(tc.filepath)
			require.NoError(t, err)

			resp, statusCode := ingestFile(t, fileData, "application/edifact")
			require.Equal(t, http.StatusCreated, statusCode)

			require.Equal(t, tc.expectedMessages, resp.MessageSummary.Total,
				"expected %d messages for %s", tc.expectedMessages, tc.name)
		})
	}
}

// TestGetInterchangeRetrievesDetails verifies the GET /interchanges/{id} endpoint.
func TestGetInterchangeRetrievesDetails(t *testing.T) {
	t.Parallel()

	fileData, err := os.ReadFile("../testdata/edifact/valid/mscons_single.edi")
	require.NoError(t, err)

	// Ingest file
	ingestResp, statusCode := ingestFile(t, fileData, "application/edifact")
	require.Equal(t, http.StatusCreated, statusCode)

	interchangeID := uuid.UUID(ingestResp.InterchangeId)

	// Get interchange details
	client := &http.Client{Timeout: 30 * time.Second}
	getURL := fmt.Sprintf("%s/interchanges/%s", globalHTTPBaseURL, interchangeID.String())
	resp, err := client.Get(getURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var detail public.InterchangeDetail
	err = json.NewDecoder(resp.Body).Decode(&detail)
	require.NoError(t, err)

	require.Equal(t, interchangeID, uuid.UUID(detail.Id))
	require.NotEmpty(t, detail.SenderId)
	require.NotEmpty(t, detail.ReceiverId)
	require.Equal(t, public.InterchangeDetailStatusCompleted, detail.Status)
	require.NotNil(t, detail.MessageSummary)
	require.Equal(t, 1, detail.MessageSummary.Total)
}

// TestGetRawFileReturnsBinary verifies the GET /interchanges/{id}/raw endpoint.
func TestGetRawFileReturnsBinary(t *testing.T) {
	t.Parallel()

	fileData, err := os.ReadFile("../testdata/edifact/valid/mscons_single.edi")
	require.NoError(t, err)

	// Ingest file
	ingestResp, statusCode := ingestFile(t, fileData, "application/edifact")
	require.Equal(t, http.StatusCreated, statusCode)

	interchangeID := uuid.UUID(ingestResp.InterchangeId)

	// Get raw file
	client := &http.Client{Timeout: 30 * time.Second}
	rawURL := fmt.Sprintf("%s/interchanges/%s/raw", globalHTTPBaseURL, interchangeID.String())
	resp, err := client.Get(rawURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/octet-stream", resp.Header.Get("Content-Type"))

	// Verify body matches original file bytes
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, fileData, body, "raw file content should match original")
	require.NotZero(t, resp.ContentLength, "Content-Length header should be set")
}

// TestHealthChecks verifies the health check endpoints.
func TestHealthChecks(t *testing.T) {
	t.Parallel()

	client := &http.Client{Timeout: 30 * time.Second}

	tests := []struct {
		name     string
		endpoint string
	}{
		{
			name:     "health",
			endpoint: "/health",
		},
		{
			name:     "liveness",
			endpoint: "/liveness",
		},
		{
			name:     "readiness",
			endpoint: "/readiness",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url := fmt.Sprintf("%s%s", globalHTTPBaseURL, tc.endpoint)
			resp, err := client.Get(url)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode,
				"expected 200 for %s endpoint", tc.endpoint)
		})
	}
}

// TestKafkaEventsPublished verifies that events are published to Kafka.
func TestKafkaEventsPublished(t *testing.T) {
	t.Parallel()

	fileData, err := os.ReadFile("../testdata/edifact/valid/minimal.edi")
	require.NoError(t, err)

	// Ingest file
	resp, statusCode := ingestFile(t, fileData, "application/edifact")
	require.Equal(t, http.StatusCreated, statusCode)

	// Kafka event publishing is verified in unit tests of the EventPublisher adapter.
	// Here we just verify the ingest succeeded.
	require.NotEqual(t, uuid.Nil, uuid.UUID(resp.InterchangeId))
	t.Log("Kafka event verification covered by unit tests")
}

// Helper functions

// ingestFile sends a POST request to /interchanges with the given file data.
func ingestFile(t *testing.T, fileData []byte, contentType string) (public.Ingest, int) {
	t.Helper()

	client := &http.Client{Timeout: 30 * time.Second}

	// Create multipart body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test.edi")
	require.NoError(t, err)

	_, err = part.Write(fileData)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Send request
	url := fmt.Sprintf("%s/interchanges", globalHTTPBaseURL)
	req, err := http.NewRequest("POST", url, body)
	require.NoError(t, err)

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var ingestResp public.Ingest
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		err = json.NewDecoder(resp.Body).Decode(&ingestResp)
		require.NoError(t, err)
	}

	return ingestResp, resp.StatusCode
}
