package main

import (
	"github.com/ctfer-io/chall-manager/sdk"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	sdk.Run(func(req *sdk.Request, resp *sdk.Response, opts ...pulumi.ResourceOption) error {
		resp.ConnectionInfo = pulumi.String("toto").ToStringOutput()
		return nil
	})
}
