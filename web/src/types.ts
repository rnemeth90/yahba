export interface RunRequest {
  url: string;
  method: string;
  requests: number;
  rps: number;
  workers: number;
  timeout: number;
  headers: string;
  body: string;
  insecure: boolean;
  keep_alive: boolean;
  http2: boolean;
  compression: boolean;
  random_user_agent: boolean;
}

export interface Sample {
  t: number;
  count: number;
  errors: number;
  avg_ms: number;
  p50_ms: number;
  p95_ms: number;
  p99_ms: number;
  status_codes: Record<string, number>;
}

export interface Summary {
  host: string;
  method: string;
  total_requests: number;
  successes: number;
  failures: number;
  duration_ms: number;
  rps: number;
  min_ms: number;
  avg_ms: number;
  p50_ms: number;
  p95_ms: number;
  p99_ms: number;
  max_ms: number;
  status_codes: Record<string, number>;
  bytes_sent: number;
  bytes_received: number;
}

export type RunStatus = 'running' | 'completed' | 'stopped' | 'aborted' | 'failed';

export interface Run {
  id: number;
  started_at: string;
  duration_ms: number;
  status: RunStatus;
  config: RunRequest;
  summary?: Summary;
}

export interface RunDetail {
  run: Run;
  samples: Sample[];
}
