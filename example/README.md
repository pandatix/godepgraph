# Example

This example shows a step-by-step workflow to generate the sample data from a Capture The Flag cybersecurity event.
Challenges comes from [here](https://github.com/nobrackets-ctf/NoBrackets-2024/tree/main/finale).

Multiple components, their deployment and tools are shared by [CTFer.io](https://ctfer.io) who sponsored the [NoBrackets](https://www.linkedin.com/showcase/nobrackets-ctf/) 2024 CTF Final round (Nov. 20th). The deployment code was used to deploy the real infrastructure of the event, but evolved through time. We will be using a recent version.

## Preamble

Requirements:
- [`docker`](https://docs.docker.com/engine/install/) ;
- [`kind`](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) ;
- [`helm`](https://helm.sh/docs/intro/install/) ;
- [`go`](https://go.dev/doc/install) ;
- [`pulumi`](https://www.pulumi.com/docs/iac/download-install/).

This experiment focuses on the potential impact of [CVE-2025-53632](https://nvd.nist.gov/vuln/detail/CVE-2025-53632), affecting the component [Chall-Manager](https://github.com/tfer-io/chall-manager).
The version we are using is patched to this vulnerability, but we will simulate it over the latest version to illustrate propagation, and on a similar symbol (same functionality, different name and prototype). Indeed, older versions had issues with OpenTelemetry support and trace propagation, so could not be used as part of this example.

To keep it short, the vulnerability is a zip-slip. Given the way Chall-Manager works internally, a tampered version of Pulumi or its providers could be overwritten in live, or a cached dependencies. This could lead to arbitrary code execution, which in turn could propagate to subsequent systems through tampered responses.

> [!WARNING]
> We won't provide an actual exploit of CVE-2025-53632, and will work on the hypothesis of one.

## Summary

This example's goal is to illustrate how a vulnerability laying in source code can propagate through Constituent Systems and affect the security posture of the overall System of Systems.

A complexity in this scenario is in the essence of the infrastructure: it is hosting a cybersecurity platform, so has been hardened to host vulnerable services adjacently to sensitive systems (e.g. the CTF platform). In this context, we also expect to evaluate the quality of the analysis through pivots by hosting components.

## Step-by-step

> [!NOTE]
> The steps required to generate the data are embedded into bash scripts.
> Documenting them more is not considered relevant, but we suggest curious readers to take a look so they deeply understand what we will be looking at.

All the following require your terminal to be in [example](/example/) context.

1.  First of all, we need to deploy the infrastructure the simulation is going to run on.
    To do so, open a terminal and run the following.
    ```bash
    ./kind.sh
    ```

    Please wait up to 5 minutes for Kind to be up & running. After this time, it should be fine passing to step 2.

2.  Then, deploy the CTF System of Systems.
    This step is optional if you already have extracted data.
    ```bash
    ./exp.sh
    ```

    At the end, it extracts all the input data for the next step into directory [`extract`](extract/).

3.  Analyze the results of step 2, i.e. creating the CDN, RDG and SIG of the simulated infrastructure.
    ```bash
    ./analysis.sh
    ```

4.  Bind Chall-Manager between the CDN, RDG and SIG infos.
    This step is required as we cannot guess what library correspond to which components, and which asset in the knowledge graphs.
    It could nonetheless leverage additional data in a CI/CD workflow to infer this binding.
    ```bash
    URL="localhost:$(cd deploy && pulumi stack output godepgraph-port)"
    go run cmd/godepgraph-cli/main.go --url $URL \
        alg4 binding create \
        --library.name github.com/ctfer-io/chall-manager \
        --library.version v0.5.1 \
        --component.name tmp-chall-manager \
        --component.version v0.5.1 \
        --asset.name chall-manager \
        --asset.version v0.5.1
    ```

5.  Then add the vulnerability in the source code.
    ```bash
    # TODO @lucas add the symbol identity
    go run cmd/godepgraph-cli/main.go --url $URL \
        alg4 vulnerability create \
        --identity "CVE-2025-53632 altered" \
        --threatens "github.com/ctfer-io/chall-manager/pkg/scenario.DecodeOCI"
    ```

6.  You can now open Neo4J in your browser and travel through the processed data, using [ciphers](https://neo4j.com/docs/getting-started/cypher/).
    ```bash
    open "http://localhost:$(cd deploy && pulumi stack output neo4j-ui-port)"
    ```

    Don't forget the settings to connect:
    - url: `bolt://$(cd deploy && pulumi stack output neo4j-user)`
    - username: `(cd deploy && pulumi stack output neo4j-user)`
    - password: `(cd deploy && pulumi stack output neo4j-pass)`
    - db name: `(cd deploy && pulumi stack output neo4j-dbname)`

## Ciphers

The following are example ciphers to use when analyzing the data.

<!-- TODO @lucas add ciphers -->

## Troubleshoot

**Q**: The `kind.sh` script does not seem to work.

**R**: Make sure your port 5000 is available (`sudo ss -laptn | grep 5000`), and no other docker container is named `registry` (`docker ps | grep registry`) else it will be erased (the same holds for kind). Also, make sure you have a stable and good internet connection to download all required Docker images. Most images are copied locally to limit bandwidth impact, but not all.

---

**Q**: I see error message on challenges failing to get created, by the end of `exp.sh`. Does it impact the results ?

**R**: Yes it has impact, but we are not focusing on the result but more on the interactions between components. Seeing errors is a sign of interactions between the components (and a bug in timeout handling in Chall-Manager), so is completly fine. Through the analysis step, we won't see much difference if there are errors or not.
