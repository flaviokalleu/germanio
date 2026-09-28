# Security policy

## Reporting a vulnerability

Please **do not open a public issue** for security problems. Report them privately through
GitHub: open the repository's **Security** tab and choose **Report a vulnerability**
(private vulnerability reporting). Include the `.ge` program or the request that reproduces
the problem, the Germanio commit or version, and what an attacker gains.

You will get an answer as soon as the maintainer can review it. Fixes are released with a
test that reproduces the problem, and the advisory credits the reporter unless they prefer
otherwise.

## Supported versions

Germanio is pre-1.0. Security fixes go to the `master` branch and to the next release; older
releases are not patched.

## Scope

In scope: the compiler, the runtime (HTTP server, authentication, sessions, authorization,
database access, uploads, Git hosting, outgoing requests), the CLI and the generated
applications' default behavior. Default settings are expected to be safe; an unsafe default
is a vulnerability.

What the runtime protects against by default, and how, is documented in
[`docs/SECURITY.md`](docs/SECURITY.md).
