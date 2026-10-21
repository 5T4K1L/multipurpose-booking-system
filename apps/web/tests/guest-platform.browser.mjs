// Live localhost test: creates and removes only its own fixtures. Requires schema 22.
import {createRequire} from 'node:module';
import {execFileSync} from 'node:child_process';
import {randomUUID,randomBytes,createHash} from 'node:crypto';
import assert from 'node:assert/strict';
import path from 'node:path';
import os from 'node:os';
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

  const base=`/api/v1/businesses/${business}`;
  const call=(m,p,d)=>request(m,base+p,d);
  const service=await call('post','/catalog/services',{name:'Calendar consultation',duration_minutes:30});
  const location=await call('post','/catalog/locations',{name:'Main location',timezone:'UTC'});
  const other=await call('post','/catalog/locations',{name:'Closed location',timezone:'Asia/Shanghai'});
  for(const loc of [location,other])await call('put',`/catalog/services/${service.id}/locations/${loc.id}`);
  const resource=await call('post','/catalog/resources',{name:'Two seats',location_id:location.id,capacity:1});
  await call('put',`/catalog/services/${service.id}/resources/${resource.id}`);
  const day=new Date(Date.now()+2*86400000), date=day.toISOString().slice(0,10);
  const second=new Date(day);second.setUTCDate(second.getUTCDate()+1);const date2=second.toISOString().slice(0,10);
  await call('put',`/catalog/locations/${location.id}/schedule`,{weekly_hours:[day,second].map(d=>({weekday:d.getUTCDay(),start_minute:540,end_minute:570})),closed_dates:[]});
  await call('put',`/catalog/resources/${resource.id}/availability-windows`,{windows:[{starts_at:date+'T00:00:00Z',ends_at:date2+'T23:59:00Z'}]});
  await call('put',`/catalog/services/${service.id}/scheduling-rules`,{buffer_before_minutes:0,buffer_after_minutes:0,minimum_lead_minutes:0});
  const review=await call('get','/publication');await call('put','/publication',{published:true,review_token:review.review_token});

  const duplicate=await context.request.post('http://localhost:8080/api/v1/businesses',{headers:{Origin:'http://localhost:3000'},data:{name:'Not allowed',timezone:'UTC'}});
  assert.equal(duplicate.status(),409);
  await page.goto(`http://localhost:3000/businesses/${business}`);
  await page.getByLabel('Service',{exact:true}).selectOption(service.id);
  await page.getByLabel('Location',{exact:true}).selectOption(location.id);
  await page.getByRole('button',{name:new RegExp('^'+date+':')}).click();
  await page.locator('button[aria-pressed]').filter({hasText:/9:00|5:00|17:00/}).first().click();
  await page.getByLabel('Your name',{exact:true}).fill('Guest Customer');
  await page.getByLabel('Contact number',{exact:true}).fill('+63 912 345 6789');
  await page.getByLabel('Email (optional)',{exact:true}).fill('guest@example.invalid');
  const confirmed=page.waitForResponse(r=>r.url().endsWith('/api/v1/guest/bookings')&&r.request().method()==='POST');
  await page.getByRole('button',{name:'Confirm booking',exact:true}).click();
  const created=await confirmed;assert.equal(created.status(),201,await created.text());
  const booking=await created.json();
  const token=await page.evaluate(()=>localStorage.getItem('booking-guest-access'));
  assert.ok(token);guestHashes.push(createHash('sha256').update(Buffer.from(token,'base64url')).digest('hex'));
  const g=async(method,suffix='',data,want=200,access=token)=>{
   const res=await guest.request[method]('http://localhost:8080/api/v1/guest/bookings'+suffix,{headers:{Origin:'http://localhost:3000','X-Booking-Access':access,'Idempotency-Key':randomUUID()},...(data===undefined?{}:{data})});
   assert.equal(res.status(),want,await res.text());return want===204?null:res.json();
  };
  await g('post','',{business_id:business,service_id:service.id,location_id:location.id,starts_at:date+'T09:00:00Z',expected_ends_at:date+'T09:30:00Z',expected_timezone:'UTC',contact:{name:'Guest',phone:'12345'}},409);
  await page.goto('http://localhost:3000/my-bookings');
  await page.getByRole('heading',{name:'Calendar consultation',exact:true}).waitFor();
  await page.getByText('Location / branch: Main location',{exact:true}).waitFor();
  await page.getByRole('button',{name:'Details',exact:true}).click();
  await page.getByRole('button',{name:'Edit contact details',exact:true}).click();
  await page.getByLabel('Your name',{exact:true}).fill('Updated Guest');
  await page.getByRole('button',{name:'Save contact details',exact:true}).click();
  await page.getByText('Contact details updated. The business can see your changes.',{exact:true}).waitFor();
  assert.equal((await call('get',`/bookings/${booking.id}`)).customer_name,'Updated Guest');
  await page.getByRole('button',{name:'Show my recovery link',exact:true}).click();
  const link=await page.getByLabel('Private link',{exact:true}).inputValue();
  const recovered=await browser.newContext();const recoveryPage=await recovered.newPage();
  await recoveryPage.goto(link);await recoveryPage.getByRole('heading',{name:'Calendar consultation',exact:true}).waitFor();
  assert.equal(new URL(recoveryPage.url()).hash,'');
  const outsider=await browser.newContext();
  assert.deepEqual(await (await outsider.request.get('http://localhost:8080/api/v1/guest/bookings')).json(),[]);
  await g('get','/'+booking.id,undefined,401,'A'.repeat(43));
  const otherToken=randomBytes(32).toString('base64url');guestHashes.push(createHash('sha256').update(Buffer.from(otherToken,'base64url')).digest('hex'));
  await g('post','',{business_id:business,service_id:service.id,location_id:location.id,starts_at:date+'T09:00:00Z',expected_ends_at:date+'T09:30:00Z',expected_timezone:'UTC',contact:{name:'Other Guest',phone:'12345'}},409,otherToken);
  await g('get','/'+booking.id,undefined,404,otherToken);
  await g('patch','/'+booking.id+'/contact',{name:'Intruder',phone:'12345'},404,otherToken);
  await g('post','/'+booking.id+'/cancel',{},404,otherToken);
  await g('delete','/'+booking.id,undefined,409,otherToken);
  assert.equal((await guest.request.get('http://localhost:8080'+base+'/bookings',{headers:{'X-Booking-Access':token}})).status(),401);

  await outsider.addCookies([{name:'booking_session',value:token,domain:'localhost',path:'/'}]);
  assert.equal((await outsider.request.get('http://localhost:8080/api/v1/auth/me')).status(),401);
  for(const width of [320,768,1280]){
   await page.setViewportSize({width,height:900});
   assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,'overflow '+width);
   await page.screenshot({path:path.join(process.env.BROWSER_OUTPUT_DIR || os.tmpdir(),`guest-bookings-${width}.png`),fullPage:true});
  }
  const moved=await g('post',`/${booking.id}/reschedule`,{starts_at:date2+'T09:00:00Z'},200);
  const movedID=moved.replacement.id;
  assert.equal((await g('get',`/${movedID}/contact`)).name,'Updated Guest');
  await page.goto(`http://localhost:3000/my-bookings/${movedID}`);
  await page.getByRole('button',{name:'Cancel booking',exact:true}).click();
  await page.getByRole('button',{name:'Confirm cancellation',exact:true}).click();
  await page.getByRole('button',{name:'Delete from my list',exact:true}).waitFor();
  page.once('dialog',d=>d.accept());
  await page.getByRole('button',{name:'Delete from my list',exact:true}).click();
  await page.getByText('Booking removed from your list.',{exact:true}).waitFor();
  await g('get','/'+movedID,undefined,404);
  assert.equal((await call('get',`/bookings/${movedID}`)).status,'cancelled');
  // A valid image remains private until publication and survives draft removal.
  const staleReview=await call('get','/publication');
  const png=await page.screenshot();
  const imageURL='http://localhost:8080'+base+'/image';
  let upload=await context.request.put(imageURL,{headers:{Origin:'http://localhost:3000','Content-Type':'image/png'},data:png});assert.ok(upload.ok(),await upload.text());
  const pubImage=`http://localhost:8080/api/v1/public/businesses/${business}/image`;
  assert.equal((await guest.request.get(pubImage)).status(),404);
  const stalePublish=await context.request.put('http://localhost:8080'+base+'/publication',{headers:{Origin:'http://localhost:3000'},data:{published:true,review_token:staleReview.review_token}});assert.equal(stalePublish.status(),409);
  const review2=await call('get','/publication');await call('put','/publication',{published:true,review_token:review2.review_token});
  assert.equal((await guest.request.get(pubImage)).status(),200);
  await call('delete','/image');assert.equal((await guest.request.get(pubImage)).status(),200);
  const review3=await call('get','/publication');await call('put','/publication',{published:true,review_token:review3.review_token});
  assert.equal((await guest.request.get(pubImage)).status(),404);
  upload=await context.request.put(imageURL,{headers:{Origin:'http://localhost:3000','Content-Type':'image/png'},data:Buffer.from('invalid')});assert.equal(upload.status(),400);
  assert.deepEqual(errors,[]);
  console.log(JSON.stringify({result:'PASS',guestCreate:true,contactEdit:true,recovery:true,isolation:true,doubleBooking:true,reschedule:true,cancel:true,delete:true,businessHistoryRetained:true,imagePublication:true,oneOwnedBusiness:true,widths:[320,768,1280]}));
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
