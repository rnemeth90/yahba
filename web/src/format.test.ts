import { describe, expect, it } from 'vitest';
import { formatBytes, formatDuration, formatMs } from './format';

describe('formatMs', () => {
  it('uses µs below 1ms', () => expect(formatMs(0.5)).toBe('500 µs'));
  it('uses ms between 1ms and 1s', () => expect(formatMs(12.34)).toBe('12.3 ms'));
  it('uses s from 1s', () => expect(formatMs(1500)).toBe('1.50 s'));
});

describe('formatDuration', () => {
  it('shows tenths of a second under a minute', () => expect(formatDuration(4200)).toBe('4.2s'));
  it('shows minutes and padded seconds', () => expect(formatDuration(65000)).toBe('1m 05s'));
});

describe('formatBytes', () => {
  it('formats bytes', () => expect(formatBytes(10)).toBe('10 B'));
  it('formats KB', () => expect(formatBytes(1536)).toBe('1.5 KB'));
  it('formats MB', () => expect(formatBytes(5 * 1024 * 1024)).toBe('5.0 MB'));
});
