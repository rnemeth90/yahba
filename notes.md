### To-Do List

#### Completed Tasks

- [x] **No Colors**: Removed ANSI color output to ensure compatibility across all environments.
- [x] **Graceful Shutdown**: Implemented a clean and controlled shutdown process.
- [x] **Refined `usage()`**: Improved the user experience for the `usage()` function.
- [x] **Report Formatting**: Enhanced the formatting of reports for better readability.
- [x] **Refactored `workerpool()`**: Broke it into smaller, more manageable functions.
- [x] **Added Documentation**: Comprehensive documentation added for the project.
- [x] **Consistent Timeouts**: Implemented uniform timeouts for HTTP clients, resolvers, etc.

#### In Progress
- [ ] **Advanced Metrics**:
  - Throughput over time
    - Measure the amount of data sent to the endpoint over a time period (i.e. bytes/second)
      - [x] Track the time taken for each request
      - [x] Track the total bytes sent per request
      - [ ] Aggregate the data. Create a new struct?
        - [ ] Starttime
        - [ ] endtime
        - [ ] # requests
        - [ ] bytes per request or total bytes
      - [ ] calculate bytes per second
      - [ ] report on the results (averages, max, min across intervals for summary metrics)
  - Concurrency breakdowns
  - Per-worker statistics
    - What can we gather per worker?


#### Pending Tasks

- [ ] **Remove Commented Code**: Clean up any unused or commented-out code.
- [ ] **Progress Bar**: Add a progress bar for visual feedback.
- [ ] **Test Server**: Build a test server for simulations.
- [ ] **Improve `server.New()`**: Ensure it returns an error when needed.
- [ ] **HTTP/3 Support**: Add support for HTTP/3.
- [ ] **Plugin System**: Enable extensibility for custom report formats, etc.
- [ ] **Rate Limiting Logic**: Implement logic for rate limiting (e.g., exponential backoffs for failed requests).
- [ ] **Implement Tests**: Write and execute tests for the project.
- [ ] **Makefile**: Create a Makefile for streamlined builds and tasks.
- [ ] **CI/CD**: Set up continuous integration and deployment pipelines.
