import {guestAccess} from "../my-bookings/guest-access";
export type Business = {
  id: string;
  name: string;
  timezone: string;
  role: string;
};
export type CatalogItem = {
  all_day?:boolean;
  duration_minutes?:number;
  id: string;
  name: string;
  role_name?: string;
  timezone?: string;
  location_id?: string;
  capacity?: number;
};
export type Booking = {
 customer_name?:string;customer_phone?:string;customer_email?:string;customer_notes?:string;
  id: string;
  service_id: string;
  location_id: string;
  staff_id: string | null;
  resource_id: string | null;
  starts_at: string;
  ends_at: string;
  busy_starts_at: string;
  busy_ends_at: string;
  status: string;
  supersedes_booking_id: string | null;
  replacement_id: string | null;
};
export type Slot = { starts_at: string; ends_at: string };
export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message);
  }
}
const messages: Record<string, string> = {
 BUSINESS_ALREADY_OWNED:"This account already owns a business. Open your workspace to manage it.",
 IMAGE_TOO_LARGE_OR_INVALID:"Choose a PNG or JPEG up to 2 MB and 6000 pixels per side.",
 GUEST_ACCESS_REQUIRED:"Open your private booking link or use the device where you booked.",
  PUBLIC_AVAILABILITY_CHANGED:
    "The business changed its service duration or location timezone. Refresh the profile after it publishes updated details.",
  PUBLICATION_TOO_LARGE:
    "This preview exceeds 1,000 active services or 2,000 linked locations. Reduce the active catalog before publishing.",
  BOOKING_CONFLICT:
    "That time is no longer available. Refresh the times and choose another.",
  BOOKING_STATE_CONFLICT:
    "This action is not available for the booking's current status or time. Refresh bookings.",
  IDEMPOTENCY_CONFLICT:
    "This request differs from an earlier attempt. Refresh the booking before trying again.",
  INVALID_PAGINATION: "The booking list changed. Refresh to start again.",
};
export async function api<T>(
  path: string,
  options: RequestInit = {},
): Promise<{ data: T; cursor: string }> {
  const guest=path.startsWith("/api/v1/customer/bookings");
  const guestToken=guest?guestAccess(options.method==="POST"):"";
  if(guest)path=path.replace("/api/v1/customer/bookings","/api/v1/guest/bookings");
  const response = await fetch(
    `${process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"}${path}`,
    {
      ...options,
      credentials: guest?"omit":(options.credentials ?? "include"),
      cache: "no-store",
      signal: options.signal ?? AbortSignal.timeout(15000),
      headers: {
        ...(options.body ? { "Content-Type": "application/json" } : {}),
        ...(guestToken?{"X-Booking-Access":guestToken}:{}),
        ...options.headers,
      },
    },
  );
  if (!response.ok) {
    const body = await response.json().catch(() => null);
    const fallback =
      response.status === 401
        ? "Please sign in to continue."
        : response.status === 403
          ? "You no longer have permission for this workspace."
          : response.status === 429
            ? "Too many attempts. Please wait before trying again."
            : response.status >= 500
              ? "The service is temporarily unavailable. You can retry the same action safely."
              : "Please check your entries and try again.";
    throw new ApiError(
      messages[body?.error?.code] ?? fallback,
      response.status,
    );
  }
  return {
    data:
      response.status === 204
        ? (undefined as T)
        : ((await response.json()) as T),
    cursor: response.headers.get("X-Next-Cursor") ?? "",
  };
}
export async function catalogAll(path: string): Promise<CatalogItem[]> {
  const items: CatalogItem[] = [];
  let cursor = "";
  for (let page = 0; page < 10; page++) {
    const result = await api<CatalogItem[]>(
      `${path}?limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
    );
    items.push(...result.data);
    cursor = result.cursor;
    if (!cursor) return items;
  }
  throw new Error(
    "This workspace is too large for the test screen. Use a smaller demo workspace.",
  );
}
// Persist only a pending request's random key, never session credentials. A reload
// after a lost response reuses it; success or a definite rejection removes it.
export async function mutate<T>(
  actor: number,
  path: string,
  body: object,
): Promise<T> {
  const payload = JSON.stringify(body);
  const storageKey = `booking-pending:${path.startsWith("/api/v1/customer/bookings")?guestAccess(true):actor}:${path}:${payload}`;
  const key = sessionStorage.getItem(storageKey) ?? crypto.randomUUID();
  sessionStorage.setItem(storageKey, key);
  try {
    const { data } = await api<T>(path, {
      method: "POST",
      body: payload,
      headers: { "Idempotency-Key": key },
    });
    sessionStorage.removeItem(storageKey);
    return data;
  } catch (error) {
    if (error instanceof ApiError && error.status < 500 && error.status !== 429)
      sessionStorage.removeItem(storageKey);
    throw error;
  }
}
export function readable(error: unknown): string {
  return error instanceof ApiError
    ? error.message
    : error instanceof Error && error.message.startsWith("This workspace")
      ? error.message
      : "Could not reach the service. Check your connection and retry.";
}
export function dateInZone(date: Date, zone: string): string {
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: zone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(date);
  const value = (type: string) =>
    parts.find((part) => part.type === type)?.value;
  return `${value("year")}-${value("month")}-${value("day")}`;
}
