package global

var (
	Version = ""
)

// Configuration holds the parameters that are shared across submodules.
type Configuration struct {
	LogLevel string

	Otlp struct {
		Tracing     bool
		ServiceName string
	}

	Neo4J struct {
		URL    string
		User   string
		Pass   string
		DBName string
	}
}

var (
	Conf Configuration
)
