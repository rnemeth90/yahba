# YAHBA - Yet Another HTTP Benchmark Application

YAHBA is a high-performance HTTP load testing tool designed to stress test your HTTP servers with customizable options like request rate, concurrency, headers, and more.

---

## Motivation

Existing HTTP benchmarking tools often lack flexibility or are overly complex for simple use cases. YAHBA was created to provide a straightforward, yet powerful tool for developers who need to quickly stress test their HTTP servers with customizable options like request rate, concurrency, headers, and payloads.

---

## Quick Start

### Using Go

```bash
go install github.com/rnemeth90/yahba@latest
```

### Build from Source

```bash
git clone https://github.com/rnemeth90/yahba.git
cd yahba
go build -o yahba .
mv yahba /usr/local/bin/
```

### Run Your First Test

```bash
yahba run --url=http://example.com --requests=100 --rps=10
```

---

## Usage

`yahba` exposes three subcommands: `run` (load test), `server` (local test HTTP server), and `tui` (interactive dashboard). All flags below apply to `yahba run`.

### Options

| Option                        | Default    | Description                                                                                                |
| ----------------------------- | ---------- | ----------------------------------------------------------------------------------------------------------- |
| `--url` or `-u`                | (required) | The target URL to stress test. Includes protocol (`http://` or `https://`).                                  |
| `--requests` or `-r`           | `4`        | Total number of requests to send.                                                                            |
| `--rps`                        | `1`        | Requests per second (RPS).                                                                                    |
| `--method` or `-m`             | `GET`      | HTTP method to use (`GET`, `POST`, `PUT`, `HEAD`, `DELETE`).                                                  |
| `--headers` or `-H`            | `""`       | Custom headers as `Key1:Value1,Key2:Value2`.                                                                  |
| `--body` or `-b`               | `""`       | Request payload (e.g., JSON or form data). Required for `POST`/`PUT`.                                         |
| `--timeout` or `-t`            | `10`       | Request timeout in seconds.                                                                                   |
| `--insecure` or `-i`           | `false`    | Disable SSL/TLS verification.                                                                                 |
| `--proxy` or `-P`              | `""`       | Proxy server in `IP:Port` format.                                                                             |
| `--proxy-user`                 | `""`       | Proxy authentication username (requires `--proxy-password`).                                                  |
| `--proxy-password`             | `""`       | Proxy authentication password (requires `--proxy-user`).                                                      |
| `--resolver`                   | `""`       | Custom DNS resolver in `IP:Port` format. Mutually exclusive with `--skip-dns`.                                 |
| `--skip-dns`                   | `false`    | Skip DNS resolution (target URL must be an IP address). Mutually exclusive with `--resolver`.                 |
| `--keep-alive` or `-k`         | `false`    | Enable HTTP keep-alive.                                                                                        |
| `--http2`                      | `false`    | Enable HTTP/2 support. Mutually exclusive with HTTP/3.                                                         |
| `--reuse-connections` or `-R`  | `false`    | Multiplex connections over a single connection (HTTP/2 only).                                                  |
| `--compression`                | `false`    | Enable HTTP compression (gzip).                                                                                |
| `--random-user-agent`          | `false`    | Randomize the `User-Agent` header per request.                                                                 |
| `--sleep` or `-s`              | `0`        | Additional sleep (in seconds) between dispatched requests.                                                     |
| `--workers` or `-w`            | `10`       | Number of concurrent workers (max `1000`).                                                                     |
| `--log-level` or `-l`          | `error`    | Logging level (`debug`, `info`, `warn`, `error`).                                                              |
| `--format` or `-f`             | `raw`      | Output format (`raw`, `json`, `yaml`).                                                                         |
| `--out`                        | `stdout`   | Output destination: `stdout` or `file`.                                                                        |
| `--filename`                   | `""`       | Output file name, used when `--out=file`.                                                                      |
| `--file-name`                  | `""`       | Path to a test definition (def) file. When set, runs the requests defined in the file instead of `--url`. See [Test Definition Files](#test-definition-files). |

### Examples

#### Test with Custom Headers

```bash
yahba run --url=https://api.example.com --headers="Authorization:******"
```

#### Send POST Requests with Payload

```bash
yahba run --url=https://api.example.com --method=POST --body='{"key":"value"}'
```

#### Use a Proxy

```bash
yahba run --url=http://example.com --proxy="http://proxy.example.com:8080"
```

#### Save Results in JSON

```bash
yahba run --url=http://example.com --format=json --out=file --filename=results.json
```

#### Run Multiple Requests from a Definition File

```bash
yahba run --file-name=requests.yaml --log-level=debug
```

> A ready-to-run sample is included at [`examples/requests.yaml`](examples/requests.yaml):
> ```bash
> yahba run --file-name=examples/requests.yaml
> ```

---

## Test Definition Files

Instead of specifying a single `--url`, you can define one or more named requests in a YAML file and pass it via `--file-name`. Requests are executed in dependency order (via `depends_on`) and results are aggregated into a single combined report.

```yaml
requests:
  - name: get-homepage
    url: https://example.com/
    method: GET
    headers:
      Authorization: "******"
    rps: 20
    requests: 500
  - name: create-user
    url: https://example.com/api/users
    method: POST
    body: '{"name":"test"}'
    headers:
      Content-Type: application/json
    rps: 5
    requests: 100
    depends_on: get-homepage   # optional sequencing
```

| Field        | Required | Default | Description                                                            |
| ------------ | -------- | ------- | ------------------------------------------------------------------------ |
| `name`       | Yes      | —       | Unique identifier for the request.                                      |
| `url`        | Yes      | —       | Target URL, must include `http://` or `https://`.                       |
| `method`     | No       | `GET`   | HTTP method (`GET`, `POST`, `PUT`, `HEAD`, `DELETE`).                    |
| `body`       | No*      | `""`    | Request payload. Required when `method` is `POST` or `PUT`.             |
| `headers`    | No       | `{}`    | Map of header key/value pairs.                                          |
| `rps`        | No       | `1`     | Requests per second for this request.                                   |
| `requests`   | No       | `1`     | Total number of requests to send for this entry.                        |
| `depends_on` | No       | `""`    | Name of another request that must complete first. Cannot form a cycle.  |

---

## Contributing

Contributions are welcome! To get started:

1. Fork the repository.
2. Create a new branch:
   ```bash
   git checkout -b feature-name
   ```
3. Make your changes and commit them:
   ```bash
   git commit -m "Description of your changes"
   ```
4. Push to your fork:
   ```bash
   git push origin feature-name
   ```
5. Open a pull request.

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
