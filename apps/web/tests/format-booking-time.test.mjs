import test from 'node:test';
import assert from 'node:assert/strict';
import {formatBookingTime} from '../app/bookings/format-booking-time.ts';
test('same local date appears once and keeps AM/PM transitions clear',()=>{
 const label=formatBookingTime('2026-10-22T00:00:00Z','2026-10-22T02:00:00Z','en-US','UTC');
 assert.equal((label.match(/Oct 22, 2026/g)||[]).length,1);
 assert.match(label,/12:00/);assert.match(label,/2:00/);assert.match(label,/AM/);
 const crossing=formatBookingTime('2026-10-22T11:00:00Z','2026-10-22T14:00:00Z','en-US','UTC');
 assert.match(crossing,/AM/);assert.match(crossing,/PM/);
});
test('different local dates are retained, using customer timezone',()=>{
 const label=formatBookingTime('2026-10-22T06:00:00Z','2026-10-22T09:00:00Z','en-US','America/Los_Angeles');
 assert.match(label,/Oct 21, 2026/);assert.match(label,/Oct 22, 2026/);
});
