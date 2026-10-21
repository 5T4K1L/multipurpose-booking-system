// Convert wall-clock input explicitly; never use the browser's timezone or
// silently normalize nonexistent/ambiguous DST times.
export function localDateTime(instant: string, zone: string): string {
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: zone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  }).formatToParts(new Date(instant));
  const get = (key: string) => parts.find((p) => p.type === key)?.value;
  return `${get("year")}-${get("month")}-${get("day")}T${get("hour")}:${get("minute")}`;
}
export function localToInstant(value: string, zone: string): string {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(value))
    throw new Error("Enter a complete date and time.");
  const guess = Date.parse(value + ":00Z");
  if (!Number.isFinite(guess)) throw new Error("Enter a valid date and time.");
  const offsets = new Set<number>();
  for (const shift of [-86400000, 0, 86400000]) {
    const instant = guess + shift;
    offsets.add(
      Date.parse(
        localDateTime(new Date(instant).toISOString(), zone) + ":00Z",
      ) - instant,
    );
  }
  const matches = [...offsets]
    .map((offset) => new Date(guess - offset).toISOString())
    .filter((instant) => localDateTime(instant, zone) === value);
  if (matches.length !== 1)
    throw new Error(
      matches.length
        ? "This time occurs twice because clocks change. Use UTC or choose a different time."
        : "This local time does not exist. Choose another time.",
    );
  return matches[0];
}
