# Custom start/end bookings

The deferred end-to-end workflow was verified on 2026-10-18 during Phase 7.9.

## Configure a service

1. Open /business and select the business.
2. Link the service to an active location and any intended staff/resources.
3. Under Scheduling, save the location opening hours and any closed dates.
   Staff/resources follow those hours. Existing bookings and capacity still apply.
4. Save the service preparation, cleanup and advance-notice rules, even if zero.
5. In Service booking mode, select Customer chooses start and end, then set minimum
   and maximum minutes. 60 minutes is 1 hour; 1440 is 1 day. Maximum is 10080 (7 days).
6. Review and publish the profile/update. Publishing saves pending workspace edits
   first and stops if a save fails. No existing business is automatically published.

The booking setup checklist links to missing configuration. It is a setup check,
not a guarantee that a particular future date has free capacity. The entire booking
and its buffers must fit opening hours across all days. Use 24-hour opening days
only if the business actually accepts continuous overnight bookings.

## Customer flow

At /businesses select the business, service, branch and available start date.
The calendar is based on the minimum length. Pick a start time and enter the end
date/time in your device timezone, then Check booking range. Only a successful
full-interval check enables confirmation. Dates/times are shown locally; the branch
calendar remains explicitly labeled. Same-day confirmation prints the date once.
A non-existent or ambiguous daylight-saving time is rejected rather than guessed.

Confirmation sends UTC instants and the expected branch timezone. The backend
rechecks current reviewed policy, valid assignment combinations, opening hours,
closures, buffers, notice and occupancy in the creation transaction. A changed
policy requires a fresh owner-reviewed publication. Conflict returns 409, not a
second reservation. Existing retry/ownership guarantees remain in force.

The saved range can be cancelled and rescheduled through customer booking details.
Rescheduling preserves its original duration. This does not offer arbitrary changes
to branch/service/assignment or a duration edit to an existing booking.

## Verification

Full Go database tests include min/max validation, stale publication, across-midnight
intervals, interior closed dates, occupancy, replay and duration preservation.
The owner-to-customer browser flow verified a 25-hour booking, duplicate overlap
rejection, cancellation and save-before-publish at mobile/tablet/desktop sizes.
See PHASE_7_CLOSURE.md for current evidence and carried-forward limitations.
