# Running `sonobuoy` Conformance Tests using Go

Our goal is to run the following command, not using the `sonobuoy` CLI, but from
Go code:

```sh
sonobuoy run --mode=certified-conformance --e2e-repo-config=conformance-image-config.yaml
```

The `e2e-repo-config` flag points to a file that contains a single line
(`dockerLibraryRegistry: mirror.gcr.io/library`), allowing us to not pull from
DockerHub.

We can use `sonobuoy gen` with the same flags to have the CLI generate the
manifests instead of applying them. This can be used to work towards manifest
parity between the outputs of the CLI and our Go code.

The first thing to note is that the `--mode` flag is merely a shortcut to
configure the `e2e` plugin. It sets the `E2E_FOCUS` and `E2E_SKIP` settings. The
equivalent command with the flags being set explicitly looks like this:

```sh
sonobuoy gen --e2e-focus='\[Conformance\]' --e2e-skip='' --e2e-repo-config=conformance-image-config.yaml
```

The `e2e_skip` flag is set to empty string, because leaving it unset triggers
the default setting, which is not what we want.

Finally, sonobuoy by default also starts the `systemd-logs` plugin, which is not
needed for Kubernetes conformance tests. We can deactivate that by specifying
the plugins explicitly:

```sh
sonobuoy gen -p=e2e --e2e-focus='\[Conformance\]' --e2e-skip='' --e2e-repo-config=conformance-image-config.yaml
```

ANd that's the final command whose output we want to match. To help check this,
use the provided `make` target:

```sh
make compare
```

This saves both outputs to temporary files and diffs them. If no ouput is shown,
that means both ways of generating the manifests produced exactly the same
output.
