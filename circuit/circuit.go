package circuit

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gowok/gowok"
)

type CircuitBreaker interface {
	Execute(f func() error) error
}

type circuitBreaker struct {
	mu           sync.Mutex
	key          string
	failureCount int
	successCount int
	state        string
	lastFailure  time.Time
	interval     time.Duration
}

// circuit is variable for storing data base on url
// current now this variabel only used by function
// SendHttpRequest
var circuit map[string]*circuitBreaker
var configCircuit configOpt
var once sync.Once

var (
	ERR_CIRCUIT_OPEN                 = fmt.Errorf("circuit breaker is open")
	ERR_CIRCUIT_INTERAL_SERVER_ERROR = fmt.Errorf("internal server error")
)

func Configure(project *gowok.Project) {

	conf, err := parseConfig(project.ConfigMap)
	if err != nil {
		slog.Warn(err.Error(), "plugin", "circuit")
		return
	}

	once.Do(func() {
		circuit = make(map[string]*circuitBreaker)
		configCircuit = conf
	})

}

func Get(name string) CircuitBreaker {
	cb, ok := circuit[name]
	if !ok {
		cb = newCircuitBreaker(name)
	}

	return cb
}

func newCircuitBreaker(key string) *circuitBreaker {
	cb := &circuitBreaker{
		key:      key,
		state:    "CLOSED",
		interval: configCircuit.duration,
	}

	circuit[key] = cb
	return cb
}

func (cb *circuitBreaker) Execute(f func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == "OPEN" {
		if time.Since(cb.lastFailure) > cb.interval {
			cb.state = "HALF-OPEN"
		} else {
			return ERR_CIRCUIT_OPEN
		}
	}

	err := f()
	if err != nil {
		cb.failureCount++
		cb.lastFailure = time.Now()
		if cb.failureCount >= configCircuit.maxFailure {
			cb.state = "OPEN"
		}

		if errors.Is(err, ERR_CIRCUIT_INTERAL_SERVER_ERROR) {
			err = nil
		}
	} else {
		cb.successCount++
		cb.failureCount = 0
		if cb.state == "HALF-OPEN" {
			cb.state = "CLOSED"
		}
	}

	if _, ok := circuit[cb.key]; ok {
		circuit[cb.key] = cb
	}
	return err
}
