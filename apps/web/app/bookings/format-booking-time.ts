// Display booking instants in the customer's timezone (browser default).
export function formatBookingTime(
  start: string,
  end: string,
  locale?: string,
  timeZone?: string,
): string {
  const from = new Date(start),
    to = new Date(end);
  const date = new Intl.DateTimeFormat(locale, {
    timeZone,
    dateStyle: "medium",
  });
  const time = new Intl.DateTimeFormat(locale, {
    timeZone,
    timeStyle: "short",
  });
  if (date.format(from) === date.format(to)) {
    return `${date.format(from)} - ${time.formatRange(from, to)}`;
  }
  return `${date.format(from)}, ${time.format(from)} - ${date.format(to)}, ${time.format(to)}`;
}
