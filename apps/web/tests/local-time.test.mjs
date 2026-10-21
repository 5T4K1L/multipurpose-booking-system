import test from "node:test";
import assert from "node:assert/strict";
import { localDateTime, localToInstant } from "../app/bookings/local-time.ts";

test("local availability uses the specified timezone, including fractional offsets", () => {
  assert.equal(
    localToInstant("2026-10-19T09:00", "Asia/Shanghai"),
    "2026-10-19T01:00:00.000Z",
  );
  assert.equal(
    localToInstant("2026-10-19T09:00", "Asia/Kathmandu"),
    "2026-10-19T03:15:00.000Z",
  );
  assert.equal(
    localDateTime("2026-10-19T16:00:00Z", "Asia/Shanghai"),
    "2026-10-20T00:00",
  );
});
test("DST gaps, folds and invalid calendar dates are not silently normalized", () => {
  assert.throws(
    () => localToInstant("2026-03-08T02:30", "America/New_York"),
    /does not exist/,
  );
  assert.throws(
    () => localToInstant("2026-11-01T01:30", "America/New_York"),
    /occurs twice/,
  );
  assert.throws(
    () => localToInstant("2026-02-30T12:00", "UTC"),
    /does not exist/,
  );
  assert.equal(
    localToInstant("2028-02-29T12:00", "UTC"),
    "2028-02-29T12:00:00.000Z",
  );
});
