package neo4j

import (
	"context"
	"sync"
	"time"

	"github.com/eapache/go-resiliency/retrier"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/sony/gobreaker/v2"
	"go.uber.org/zap"
)

type Manager struct {
	mu     sync.RWMutex
	driver neo4j.DriverWithContext
	config Config

	breaker *gobreaker.CircuitBreaker[any]
	retrier *retrier.Retrier
}

func NewManager(config Config) *Manager {
	cbSettings := gobreaker.Settings{
		Name:          "neo4j circuit breaker",
		OnStateChange: config.CBOnStateChange,
	}

	return &Manager{
		config:  config,
		breaker: gobreaker.NewCircuitBreaker[any](cbSettings),
		retrier: retrier.New(retrier.ExponentialBackoff(3, 300*time.Millisecond), nil),
	}
}

type Config struct {
	URL      string
	Username string
	Password string
	DBName   string

	CBOnStateChange func(name string, from, to gobreaker.State)
}

func (m *Manager) NewSession(ctx context.Context) (neo4j.SessionWithContext, error) {
	d, err := m.getDriver(ctx)
	if err != nil {
		return nil, err
	}
	return d.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: m.config.DBName,
	}), nil
}

func (m *Manager) getDriver(ctx context.Context) (neo4j.DriverWithContext, error) {
	m.mu.RLock()
	driver := m.driver
	m.mu.RUnlock()

	if driver != nil {
		if err := driver.VerifyConnectivity(ctx); err == nil {
			return driver, nil
		}
	}

	return m.recreateDriver(ctx)
}

func (m *Manager) recreateDriver(ctx context.Context) (neo4j.DriverWithContext, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.driver != nil {
		if err := m.driver.VerifyConnectivity(ctx); err == nil {
			return m.driver, nil
		}
		_ = m.driver.Close(ctx)
		m.driver = nil
	}

	driver, err := neo4j.NewDriverWithContext(
		m.config.URL,
		neo4j.BasicAuth(m.config.Username, m.config.Password, ""),
	)
	if err != nil {
		return nil, err
	}

	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(ctx)
		return nil, err
	}

	m.driver = driver
	return driver, nil
}

func (m *Manager) Execute(ctx context.Context, operation func(context.Context, neo4j.DriverWithContext) error) error {
	return m.retrier.Run(func() error {
		_, err := m.breaker.Execute(func() (any, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			driver, err := m.getDriver(timeoutCtx)
			if err != nil {
				zap.L().Error("Failed to get Neo4j driver", zap.Error(err))
				return nil, err
			}

			err = operation(timeoutCtx, driver)
			return nil, err
		})

		return err
	})
}

func (m *Manager) Healthcheck(ctx context.Context) error {
	d, err := m.getDriver(ctx)
	if err != nil {
		return err
	}
	return d.VerifyConnectivity(ctx)
}
