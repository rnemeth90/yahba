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
yahba --url=http://example.com --requests=100 --rps=10
```

---

## Usage

### Options

| Option               | Default    | Description                                                                 |
| -------------------- | ---------- | --------------------------------------------------------------------------- |
| `--url` or `-u`      | (required) | The target URL to stress test. Includes protocol (`http://` or `https://`). |
| `--requests` or `-r` | `4`        | Total number of requests to send.                                           |
| `--rps`              | `1`        | Requests per second (RPS).                                                  |
| `--method` or `-m`   | `GET`      | HTTP method to use (`GET`, `POST`, etc.).                                   |
| `--headers` or `-H`  | `""`       | Custom headers as `Key1:Value1,Key2:Value2`.                                |
| `--body` or `-b`     | `""`       | Request payload (e.g., JSON or form data).                                  |
| `--timeout` or `-t`  | `10`       | Request timeout in seconds.                                                 |
| `--insecure` or `-i` | `false`    | Disable SSL/TLS verification.                                               |
| `--proxy` or `-P`    | `""`       | Proxy server in `IP:Port` format.                                           |
| `--output-format`    | `raw`      | Output format (`raw`, `json`, `yaml`).                                      |
| `--out`              | `stdout`   | File path for saving results.                                               |

### Examples

#### Test with Custom Headers

```bash
yahba --url=https://api.example.com --headers="Authorization:Bearer abc123,Content-Type:application/json"
```

#### Send POST Requests with Payload

```bash
yahba --url=https://api.example.com --method=POST --body='{"key":"value"}'
```

#### Use a Proxy

```bash
yahba --url=http://example.com --proxy="http://proxy.example.com:8080"
```

#### Save Results in JSON

```bash
yahba --url=http://example.com --output-format=json > results.json
```

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
