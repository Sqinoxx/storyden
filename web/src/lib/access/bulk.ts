export type BulkResult = { ok: number; failed: number };

export async function runBulk<T>(
  items: T[],
  fn: (item: T) => Promise<unknown>,
  concurrency = 4,
): Promise<BulkResult> {
  let ok = 0;
  let failed = 0;
  let next = 0;

  async function worker() {
    while (next < items.length) {
      const item = items[next++] as T;
      try {
        await fn(item);
        ok++;
      } catch {
        failed++;
      }
    }
  }

  const workers = Math.max(1, Math.min(concurrency, items.length));
  await Promise.all(Array.from({ length: workers }, worker));

  return { ok, failed };
}
