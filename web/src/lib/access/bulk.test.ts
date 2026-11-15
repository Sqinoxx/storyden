import { test } from "uvu";
import * as assert from "uvu/assert";

import { runBulk } from "./bulk";

test("runBulk counts successes and failures", async () => {
  const seen: number[] = [];
  const result = await runBulk([1, 2, 3, 4, 5], async (n) => {
    seen.push(n);
    if (n % 2 === 0) throw new Error("even");
  });

  assert.equal(result, { ok: 3, failed: 2 });
  assert.equal([...seen].sort(), [1, 2, 3, 4, 5]);
});

test("runBulk never exceeds the concurrency limit", async () => {
  let running = 0;
  let peak = 0;

  await runBulk(
    Array.from({ length: 10 }, (_, i) => i),
    async () => {
      running++;
      peak = Math.max(peak, running);
      await new Promise((r) => setTimeout(r, 5));
      running--;
    },
    3,
  );

  assert.is(peak, 3);
});

test("runBulk handles an empty list", async () => {
  assert.equal(await runBulk([], async () => {}), { ok: 0, failed: 0 });
});

test.run();
