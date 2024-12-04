input:

url
proxy
proxyuser
proxypassword
insecure
keepalive
compression
sleep
requests
timeout
rps
method
headers
resolvers
body
log package
http1.1

verbose (logging package)
output

todo:
[ ] no colors... (some environments obviously will not support ANSI colors, leading to problems when reading output)
[ ] graceful shutdown
[ ] refine usage()
[ ] improve report formatting
[ ] Refactor workerpool() into smaller functions
[ ] add docs
[ ] remove commented code
[ ] implement consistent timeouts (httpclient, resolvers, etc.)

[ ] advanced metrics (throughput over time, concurrency breakdowns, per-worker statistics)
[ ] progress bar?
[ ] test server

[ ] http3 support
[ ] plugin system for extensibility? (custom report formats, etc.)
[ ] rate limiting logic (exponential backoffs for failed requests?)

[ ] implement tests

[ ] makefile
[ ] CI/CD
