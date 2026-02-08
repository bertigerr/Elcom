import path from 'node:path';
import dotenv from 'dotenv';

dotenv.config();

const cwd = process.cwd();

function asNumber(value: string | undefined, fallback: number): number {
  if (!value) {
    return fallback;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function asBool(value: string | undefined, fallback: boolean): boolean {
  if (value == null) {
    return fallback;
  }
  const normalized = value.trim().toLowerCase();
  if (['1', 'true', 'yes', 'on'].includes(normalized)) {
    return true;
  }
  if (['0', 'false', 'no', 'off'].includes(normalized)) {
    return false;
  }
  return fallback;
}

export const config = {
  dbPath: process.env.DB_PATH ?? path.join(cwd, 'data', 'app.db'),
  rawMailDir: process.env.MAIL_RAW_DIR ?? path.join(cwd, 'data', 'raw'),
  outputDir: process.env.OUTPUT_DIR ?? path.join(cwd, 'out'),

  elcomApiBaseUrl: process.env.ELCOM_API_BASE_URL ?? 'https://online.el-com.ru/api/v1',
  elcomApiToken: process.env.ELCOM_API_TOKEN ?? '',
  elcomRateLimitRps: asNumber(process.env.ELCOM_RATE_LIMIT_RPS, 5),
  elcomTimeoutMs: asNumber(process.env.ELCOM_TIMEOUT_MS, 30000),
  incrementalLookbackHours: asNumber(process.env.ELCOM_INCREMENTAL_HOURS, 24),
  incrementalLookbackDays: asNumber(process.env.ELCOM_INCREMENTAL_DAYS, 2),

  matchOkThreshold: asNumber(process.env.MATCH_OK_THRESHOLD, 0.9),
  matchReviewThreshold: asNumber(process.env.MATCH_REVIEW_THRESHOLD, 0.72),
  matchGapThreshold: asNumber(process.env.MATCH_GAP_THRESHOLD, 0.08),

  gmailClientId: process.env.GMAIL_CLIENT_ID ?? '',
  gmailClientSecret: process.env.GMAIL_CLIENT_SECRET ?? '',
  gmailRedirectUri: process.env.GMAIL_REDIRECT_URI ?? 'https://developers.google.com/oauthplayground',
  gmailRefreshToken: process.env.GMAIL_REFRESH_TOKEN ?? '',

  imapHost: process.env.IMAP_HOST ?? '',
  imapPort: asNumber(process.env.IMAP_PORT, 993),
  imapSecure: asBool(process.env.IMAP_SECURE, true),
  imapUser: process.env.IMAP_USER ?? '',
  imapPassword: process.env.IMAP_PASSWORD ?? '',
  imapMarkSeen: asBool(process.env.IMAP_MARK_SEEN, false),

  mailListenerProvider: (process.env.MAIL_LISTENER_PROVIDER ?? 'gmail') as 'gmail' | 'imap',
  mailListenerLabel: process.env.MAIL_LISTENER_LABEL ?? 'INBOX',
  mailListenerIntervalSec: asNumber(process.env.MAIL_LISTENER_INTERVAL_SEC, 30),
  mailListenerFetchMax: asNumber(process.env.MAIL_LISTENER_FETCH_MAX, 20),
  mailListenerProcessBatch: asNumber(process.env.MAIL_LISTENER_PROCESS_BATCH, 20),
  mailListenerAutoExport: asBool(process.env.MAIL_LISTENER_AUTO_EXPORT, true),
};

export function requireEnv(value: string, name: string): string {
  if (!value) {
    throw new Error(`Missing required env var: ${name}`);
  }
  return value;
}
