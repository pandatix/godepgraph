package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v3"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/pandatix/godepgraph/global"
	"github.com/pandatix/godepgraph/server"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
)

func main() {
	app := &cli.Command{
		Name:  "GoDepGraph",
		Usage: "Toolbox for Reconstructing Systems-of-Systems architectures towards analyzing cascading attacks, in Go.",
		Flags: []cli.Flag{
			cli.VersionFlag,
			cli.HelpFlag,
			&cli.IntFlag{
				Name:     "port",
				Aliases:  []string{"p"},
				Sources:  cli.EnvVars("PORT"),
				Category: "global",
				Value:    8080,
				Usage:    "Define the API server port to listen on (gRPC+HTTP).",
			},
			&cli.BoolFlag{
				Name:     "swagger",
				Sources:  cli.EnvVars("SWAGGER"),
				Category: "global",
				Value:    false,
				Usage:    "If set, turns on the API gateway swagger on `/swagger`.",
			},
			&cli.StringFlag{
				Name:     "log-level",
				Sources:  cli.EnvVars("LOG_LEVEL"),
				Category: "global",
				Value:    "info",
				Action: func(_ context.Context, _ *cli.Command, lvl string) error {
					_, err := zapcore.ParseLevel(lvl)
					return err
				},
				Destination: &global.Conf.LogLevel,
				Usage:       "Use to specify the level of logging.",
			},
			&cli.BoolFlag{
				Name:        "otlp.tracing",
				Sources:     cli.EnvVars("OTLP_TRACING"),
				Category:    "otlp",
				Destination: &global.Conf.Otlp.Tracing,
				Usage:       "If set, turns on tracing through OpenTelemetry (see https://opentelemetry.io for more info).",
			},
			&cli.StringFlag{
				Name:        "otlp.service-name",
				Sources:     cli.EnvVars("OTLP_SERVICE_NAME"),
				Category:    "otlp",
				Value:       "godepgraph",
				Destination: &global.Conf.Otlp.ServiceName,
				Usage:       "Override the service name. Useful when deploying multiple instances to filter signals.",
			},
			&cli.StringFlag{
				Name:        "neo4j.uri",
				Sources:     cli.EnvVars("NEO4J_URI"),
				Category:    "neo4j",
				Required:    true,
				Destination: &global.Conf.Neo4J.URL,
				Usage:       "The Neo4J URI to export data. Example: bolt://localhost:7687",
			},
			&cli.StringFlag{
				Name:        "neo4j.user",
				Sources:     cli.EnvVars("NEO4J_USER"),
				Category:    "neo4j",
				Required:    true,
				Destination: &global.Conf.Neo4J.User,
				Usage:       "The Neo4J user.",
			},
			&cli.StringFlag{
				Name:        "neo4j.pass",
				Sources:     cli.EnvVars("NEO4J_PASS"),
				Category:    "neo4j",
				Required:    true,
				Destination: &global.Conf.Neo4J.Pass,
				Usage:       "The Neo4J pass.",
			},
			&cli.StringFlag{
				Name:        "neo4j.dbname",
				Sources:     cli.EnvVars("NEO4J_DBNAME"),
				Category:    "neo4j",
				Value:       "godepgraph",
				Destination: &global.Conf.Neo4J.DBName,
				Usage:       "The Neo4J database name in which to export data.",
			},
		},
		Action:  run,
		Version: version,
		Metadata: map[string]any{
			"version": version,
			"commit":  commit,
			"date":    date,
			"builtBy": builtBy,
		},
	}

	ctx := context.Background()
	if err := app.Run(ctx, os.Args); err != nil {
		global.Log().Error(ctx, "fatal error",
			zap.Error(err),
		)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	// Pre-flight global configuration
	global.Version = version

	port := cmd.Int("port")
	sw := cmd.Bool("swagger")

	// Initialize tracing and handle the tracer provider shutdown
	if global.Conf.Otlp.Tracing {
		// Set up OpenTelemetry.
		otelShutdown, err := global.SetupOtelSDK(ctx)
		if err != nil {
			return err
		}
		// Handle shutdown properly so nothing leaks.
		defer func() {
			err = multierr.Append(err, otelShutdown(ctx))
		}()
	}

	logger := global.Log()
	logger.Info(ctx, "starting API server",
		zap.Int("port", port),
		zap.Bool("swagger", sw),
		zap.Bool("tracing", global.Conf.Otlp.Tracing),
	)

	// Create context that listens for the interrupt signal from the OS
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Prepare Neo4J
	neo4jman := global.GetNeo4JManager()
	if err := global.ExecuteNeo4JInitializers(ctx, neo4jman); err != nil {
		return errors.Wrap(err, "executing neo4j initializers")
	}

	// Launch API server
	srv := server.NewServer(server.Options{
		Port:    port,
		Swagger: sw,
	})
	if err := srv.Run(ctx); err != nil {
		return err
	}

	// Listen for the interrupt signal
	<-ctx.Done()

	// Restore default behavior on the interrupt signal
	stop()
	logger.Info(ctx, "shutting down gracefully")

	return nil
}
