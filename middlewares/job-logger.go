package middlewares

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/octoblu/tattle/logentry"
)

// JobLogger holds the oxy circuit breaker.
type JobLogger struct {
	redisChannel chan []byte
	router       *mux.Router
}

// NewJobLogger returns a new JobLogger.
func NewJobLogger(redisURI, queueName string, router *mux.Router) *JobLogger {
	redisChannel := make(chan []byte)
	go runLogger(redisURI, queueName, redisChannel)

	return &JobLogger{redisChannel, router}
}

func (jobLogger *JobLogger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	startTime := time.Now()
	redisChannel := jobLogger.redisChannel

	backendName := "unknown"

	routeMatch := mux.RouteMatch{}
	if jobLogger.router.Match(r, &routeMatch) {
		backendName = routeMatch.Route.GetName()
	}

	secret := &SecretRapper{rw, redisChannel, startTime, backendName}
	next(secret, r)
}

// SecretRapper wraps in silence
type SecretRapper struct {
	rw           http.ResponseWriter
	redisChannel chan []byte
	startTime    time.Time
	backendName  string
}

// Header returns the header map that will be sent by
// WriteHeader
func (secretRapper *SecretRapper) Header() http.Header {
	return secretRapper.rw.Header()
}

// Write writes the data to the connection as part of an HTTP reply.
func (secretRapper *SecretRapper) Write(data []byte) (int, error) {
	return secretRapper.rw.Write(data)
}

// WriteHeader sends an HTTP response header with status code.
func (secretRapper *SecretRapper) WriteHeader(statusCode int) {
	secretRapper.logTheEntry(statusCode)
	secretRapper.rw.WriteHeader(statusCode)
}

func (secretRapper *SecretRapper) logTheEntry(statusCode int) {
	elapsedTimeNano := time.Now().UnixNano() - secretRapper.startTime.UnixNano()
	elapsedTime := int(elapsedTimeNano / 1000000)

	logEntry := logentry.New("metric:traefik", "http", secretRapper.backendName, "anonymous", statusCode, elapsedTime)
	logEntryBytes, err := json.Marshal(logEntry)
	logError("NewJobLogger failed: %v\n", err)

	if err != nil {
		return
	}

	select {
	case secretRapper.redisChannel <- logEntryBytes:
	default:
		fmt.Fprintln(os.Stderr, "Redis not ready, skipping logging")
	}
}

func logError(fmtMessage string, err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, fmtMessage, err.Error())
}

func runLogger(redisURI, queueName string, logChannel chan []byte) {
	redisConn, err := redis.DialURL(redisURI)
	logError("redis.DialURL Failed: %v\n", err)

	for {
		logEntryBytes := <-logChannel
		_, err = redisConn.Do("lpush", queueName, logEntryBytes)
		logError("Redis LPUSH failed: %v\n", err)
	}
}
