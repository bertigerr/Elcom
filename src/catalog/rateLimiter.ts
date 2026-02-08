export class RateLimiter {
  private nextAllowedAt = 0;

  constructor(private readonly requestsPerSecond: number) {
    if (!Number.isFinite(requestsPerSecond) || requestsPerSecond <= 0) {
      throw new Error('requestsPerSecond must be positive');
    }
  }

  async waitTurn(): Promise<void> {
    const intervalMs = Math.ceil(1000 / this.requestsPerSecond);
    const now = Date.now();
    const scheduled = Math.max(now, this.nextAllowedAt);
    this.nextAllowedAt = scheduled + intervalMs;
    const delay = scheduled - now;
    if (delay > 0) {
      await new Promise((resolve) => setTimeout(resolve, delay));
    }
  }
}
