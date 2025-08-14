package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"

	"github.com/docker/buildx/builder"
	"github.com/docker/buildx/util/imagetools"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/slsa-framework/slsa-verifier/v2/cli/slsa-verifier/verify"
)

func main() {
	// From an OCI ref...
	ref := "ctferio/chall-manager:v0.3.3"

	// Get its digest without pulling it
	dig, err := crane.Digest(ref)
	if err != nil {
		log.Fatal(err)
	}

	// Get its config to look for opencontainers labels
	out, err := crane.Config(ref)
	if err != nil {
		log.Fatal(err)
	}

	type OCIConfig struct {
		Config struct{ Labels map[string]string }
		// others are ignored
	}
	var conf OCIConfig
	if err := json.Unmarshal(out, &conf); err != nil {
		log.Fatal(err)
	}

	source, ok := conf.Config.Labels["org.opencontainers.image.source"]
	if !ok || (ok && source == "") {
		log.Fatalf("Label %s not found or empty", "org.opencontainers.image.source")
	}
	u, err := url.Parse(source)
	if err != nil {
		log.Fatal(err)
	}
	source = u.Host + u.Path
	version, ok := conf.Config.Labels["org.opencontainers.image.version"]
	if !ok || (ok && version == "") {
		log.Fatal("Label %s not found or empty", "org.opencontainers.image.version")
	}

	// Verify SLSA
	cmd := verify.VerifyImageCommand{
		SourceURI: source,
		SourceTag: &version,
	}
	if _, err := cmd.Exec(context.Background(), []string{
		fmt.Sprintf("%s@%s", ref, dig),
	}); err != nil {
		log.Fatal(err)
	}

	// Extract SBOM -> CDN
	sbom, err := getSBOM(ref)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("sbom: %s\n", sbom)
}

func getSBOM(ref string) ([]byte, error) {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		return nil, err
	}
	if err := dockerCli.Initialize(flags.NewClientOptions()); err != nil {
		return nil, err
	}

	b, err := builder.New(dockerCli,
		builder.WithName(""),
		builder.WithSkippedValidation(),
	)
	if err != nil {
		return nil, err
	}
	imageopt, err := b.ImageOpt()
	if err != nil {
		return nil, err
	}

	p, err := imagetools.NewPrinter(context.Background(), imageopt, ref, "{{json .SBOM}}")
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	if err := p.Print(false, buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
