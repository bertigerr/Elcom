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
};

export function requireEnv(value: string, name: string): string {
  if (!value) {
    throw new Error(`Missing required env var: ${name}`);
  }
  return value;
}
