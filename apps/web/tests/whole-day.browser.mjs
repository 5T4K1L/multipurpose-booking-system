// Live localhost test: creates and removes only its own fixtures. Requires schema 23.
import {createRequire} from 'node:module';
import {execFileSync} from 'node:child_process';
import {randomUUID,createHash} from 'node:crypto';
import assert from 'node:assert/strict';
const load=createRequire(import.meta.url);
const {chromium}=load(process.env.PLAYWRIGHT_MODULE || 'playwright-core');
(async()=>{
 const browser=await chromium.launch({executablePath:process.env.CHROME_PATH || 'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true});
 const context=await browser.newContext();
 const guest=await browser.newContext({viewport:{width:1280,height:900}});
 const page=await guest.newPage();page.setDefaultTimeout(30000);
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 const email=`guest-owner-${Date.now()}@example.invalid`, password=randomUUID()+'Aa1!';
 let actor,business;const guestHashes=[];
 const request=async(method,url,data)=>{
  const res=await context.request[method]('http://localhost:8080'+url,{headers:{Origin:'http://localhost:3000','Idempotency-Key':randomUUID()},...(data===undefined?{}:{data})});
  if(!res.ok())throw Error(`${method} ${url}: ${res.status()} ${await res.text()}`);
  return res.status()===204?null:res.json();
 };
 try {
  await request('post','/api/v1/auth/register',{username:'Guest fixture owner',email,password});
  await request('post','/api/v1/auth/login',{email,password});
  actor=(await request('get','/api/v1/auth/me')).user.id;
  const ownerPage=await context.newPage();
  await ownerPage.goto('http://localhost:3000/business');
  await ownerPage.getByLabel('Business name',{exact:true}).fill('Calendar fixture '+Date.now());
  await ownerPage.getByLabel('Business timezone',{exact:true}).selectOption('UTC');
  await ownerPage.getByLabel('Short introduction',{exact:true}).fill('Book a consultation with our team.');
  await ownerPage.getByLabel('Business contact email',{exact:true}).fill(email);
  await ownerPage.getByLabel('Business phone',{exact:true}).fill('+63 912 345 6789');
  const creating=ownerPage.waitForResponse(r=>r.url().endsWith('/api/v1/businesses')&&r.request().method()==='POST');
  await ownerPage.getByRole('button',{name:'Create business',exact:true}).click();
  const businessResponse=await creating;assert.equal(businessResponse.status(),201,await businessResponse.text());
  business=(await businessResponse.json()).id;
  await ownerPage.getByText('Your workspace:',{exact:false}).waitFor();
  assert.equal(await ownerPage.getByRole('button',{name:'Create business',exact:true}).count(),0);


  const base=`/api/v1/businesses/${business}`,call=(m,p,d)=>request(m,base+p,d);
  const service=await call('post','/catalog/services',{name:'Whole-day studio',all_day:true});assert.equal(service.duration_minutes,1440);assert.equal(service.all_day,true);
  const location=await call('post','/catalog/locations',{name:'Main location',timezone:'UTC'});
  await call('put',`/catalog/services/${service.id}/locations/${location.id}`);
  const resource=await call('post','/catalog/resources',{name:'Studio',location_id:location.id,capacity:1});await call('put',`/catalog/services/${service.id}/resources/${resource.id}`);
  const hours=Array.from({length:7},(_,weekday)=>({weekday,start_minute:540,end_minute:1020}));
  await call('put',`/catalog/locations/${location.id}/schedule`,{weekly_hours:hours,closed_dates:[]});
  const review=await call('get','/publication');assert.equal(review.profile.services[0].all_day,true);await call('put','/publication',{published:true,review_token:review.review_token});
  const date=new Date(Date.now()+2*86400000).toISOString().slice(0,10);
  await page.goto(`http://localhost:3000/businesses/${business}`);
  await page.locator(`[data-service-id="${service.id}"]`).click();
  await page.getByRole('button',{name:new RegExp('^'+date+':')}).click();
  await page.getByRole('group',{name:'Available start times'}).getByRole('button').first().click();
  await page.getByText('Whole day',{exact:true}).last().waitFor();
  await page.getByLabel('Your name',{exact:true}).fill('Whole day guest');await page.getByLabel('Contact number',{exact:true}).fill('123456789');
  const created=page.waitForResponse(r=>r.url().endsWith('/api/v1/guest/bookings')&&r.request().method()==='POST');await page.getByRole('button',{name:'Confirm booking',exact:true}).click();
  const response=await created;assert.equal(response.status(),201,await response.text());const booking=await response.json();assert.equal(Date.parse(booking.ends_at)-Date.parse(booking.starts_at),28800000);
  const token=await page.evaluate(()=>localStorage.getItem('booking-guest-access'));guestHashes.push(createHash('sha256').update(Buffer.from(token,'base64url')).digest('hex'));
  const g=async(method,suffix,data,want)=>{const res=await guest.request[method]('http://localhost:8080/api/v1/guest/bookings'+suffix,{headers:{Origin:'http://localhost:3000','X-Booking-Access':token,'Idempotency-Key':randomUUID()},data});assert.equal(res.status(),want,await res.text());return res.status()===204?null:res.json();};
  const body={business_id:business,service_id:service.id,location_id:location.id,starts_at:booking.starts_at,expected_ends_at:booking.ends_at,expected_timezone:'UTC',contact:{name:'Duplicate',phone:'12345'}};
  await g('post','',body,409);
  await g('post','',{...body,expected_ends_at:new Date(Date.parse(booking.starts_at)+3600000).toISOString()},409);
  const available=await request('get',`/api/v1/public/businesses/${business}/services/${service.id}/availability?location_id=${location.id}&date=${date}`);assert.equal(available.slots.length,0);
  const moved=await g('post',`/${booking.id}/reschedule`,{starts_at:new Date(Date.parse(booking.starts_at)+86400000).toISOString()},200);assert.equal(Date.parse(moved.replacement.ends_at)-Date.parse(moved.replacement.starts_at),28800000);
  await g('post',`/${moved.replacement.id}/cancel`,{},200);
  await ownerPage.goto('http://localhost:3000/business/bookings');await ownerPage.getByRole('button',{name:'Delete from business list',exact:true}).click();await ownerPage.getByRole('button',{name:'Confirm delete',exact:true}).click();await ownerPage.getByText('Cancelled booking removed from your business list. Customer history is kept.',{exact:true}).waitFor();assert.equal((await call('get','/bookings')).some(b=>b.id===moved.replacement.id),false);await g('get',`/${moved.replacement.id}`,undefined,200);
  const restored=await request('get',`/api/v1/public/businesses/${business}/services/${service.id}/availability?location_id=${location.id}&date=${date}`);assert.ok(restored.slots.length>0);
  await call('put',`/catalog/locations/${location.id}/schedule`,{weekly_hours:hours.map(h=>({...h,start_minute:540,end_minute:1020})),closed_dates:[]});
  const closed=await request('get',`/api/v1/public/businesses/${business}/services/${service.id}/availability?location_id=${location.id}&date=${date}`);assert.ok(closed.slots.length>0);for(const slot of closed.slots){const h=new Date(slot.starts_at).getUTCHours();assert.ok(h>=9&&h<17);assert.equal(Date.parse(slot.ends_at)-Date.parse(slot.starts_at),28800000);}
  assert.deepEqual(errors,[]);console.log('PASS live whole-day publication, customer booking, opening-to-closing hours, conflict, forged shorter duration rejection, reschedule, cancellation, same-day opening and closing.');
 } finally {
  if(actor){
   if(!Number.isSafeInteger(actor)||actor<=0)throw Error('Unsafe actor');
   let sql='BEGIN;\n';
   if(business){
    if(!/^[a-f0-9-]{36}$/.test(business))throw Error('Unsafe business');
    for(const table of ['booking_changes','bookings','staff_availability_windows','resource_availability_windows','location_weekly_hours','location_closed_dates','service_scheduling_rules','service_locations','service_staff','service_resources','staff_locations','resources','staff_profiles','services','locations'])sql+=`DELETE FROM ${table} WHERE business_id='${business}';\n`;
    sql+=`DELETE FROM businesses WHERE id='${business}';\n`;
   }
   for(const table of ['user_sessions','user_platform_roles','user_credentials'])sql+=`DELETE FROM ${table} WHERE user_id=${actor};\n`;
   sql+=`DELETE FROM users WHERE id=${actor} AND email='${email}';\n`;
   for(const hash of guestHashes)sql+=`DELETE FROM users WHERE email='${hash}@guest.invalid';\n`;
   sql+='COMMIT;';
   execFileSync(process.env.DOCKER_PATH || 'docker',['exec','-i','multipurpose-booking-system','psql','-v','ON_ERROR_STOP=1','-U','booking_user','-d','booking_db'],{input:sql,stdio:['pipe','pipe','pipe']});
  }
  await browser.close();
 }
})().catch(e=>{console.error(e);process.exit(1)});
