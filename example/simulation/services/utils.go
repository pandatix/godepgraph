package services

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// parseURLPort parses the input endpoint formatted as a URL to return its port.
// Example:
//
//	"http://some.thing:port" -> port
func parseURLPort(edp pulumi.StringOutput) pulumi.IntOutput {
	return edp.ToStringOutput().ApplyT(func(edp string) (int, error) {
		u, err := url.Parse(edp)
		if err != nil {
			return 0, errors.Wrapf(err, "parsing endpoint %s as a URL", edp)
		}
		p, err := strconv.Atoi(u.Port())
		if err != nil {
			return 0, errors.Wrapf(err, "parsing endpoint %s for port", edp)
		}
		return p, nil
	}).(pulumi.IntOutput)
}

// parsePort cuts the input endpoint to return its port.
// Example: some.thing:port -> port
func parsePort(edp pulumi.StringInput) pulumi.IntOutput {
	return edp.ToStringOutput().ApplyT(func(edp string) (int, error) {
		_, pStr, _ := strings.Cut(edp, ":")
		p, err := strconv.Atoi(pStr)
		if err != nil {
			return 0, errors.Wrapf(err, "parsing endpoint %s for port", edp)
		}
		return p, nil
	}).(pulumi.IntOutput)
}
