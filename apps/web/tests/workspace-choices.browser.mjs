import {createRequire} from 'node:module';
import assert from 'node:assert/strict';
import path from 'node:path';
import os from 'node:os';
const {chromium}=createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE||'playwright-core');
const browser=await chromium.launch({executablePath:'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true});
try {
 const page=await browser.newPage();const errors=[];page.on('pageerror',e=>{errors.push(e.message);console.log('PAGE ERROR',e.message)});
 const business={id:'biz',name:'Three Studios',timezone:'UTC',role:'BUSINESS_OWNER'};
 let locations=[{id:'loc',name:'Main branch',timezone:'UTC'},{id:'other',name:'Garden branch',timezone:'UTC'}];let linked=['loc'],deletes=0,createdService=null;
 await page.route('**/api/v1/**',async route=>{const req=route.request(),u=new URL(req.url()),p=u.pathname;let data={},status=200;
 if(p.endsWith('/auth/me'))data={user:{id:1,email:'fixture@example.invalid'}};
 else if(p==='/api/v1/businesses')data=[business];
 else if(p==='/api/v1/businesses/biz')data=business;
 else if(p.endsWith('/catalog/locations/other')&&req.method()==='DELETE'){deletes++;locations=locations.filter(l=>l.id!=='other');status=204;}
 else if(p.endsWith('/catalog/locations/loc')&&req.method()==='DELETE'){status=409;data={error:{code:'CATALOG_CONFLICT'}};}
 else if(p.endsWith('/catalog/locations'))data=locations;
 else if(p.endsWith('/catalog/services')&&req.method()==='POST'){createdService=req.postDataJSON();data={id:'whole',...createdService};status=201;}
 else if(p.endsWith('/catalog/services'))data=[{id:'svc',name:'Photography',duration_minutes:60},{id:'svc2',name:'Portraits',duration_minutes:30}];
 else if(p.endsWith('/services/svc/locations'))data=linked;
 else if(p.endsWith('/services/svc2/locations'))data=[];
 else if(p.endsWith('/services/svc/locations/other')){linked=req.method()==='PUT'?['loc','other']:['loc'];status=204;}
 else if(p.endsWith('/schedule'))data={timezone:'UTC',weekly_hours:[{weekday:1,start_minute:540,end_minute:1020}],closed_dates:[]};
 else if(p.endsWith('/booking-options'))data={enabled:false,minimum_minutes:30,maximum_minutes:120};
 else if(p.endsWith('/scheduling-rules'))data={configured:false,buffer_before_minutes:0,buffer_after_minutes:0,minimum_lead_minutes:0};
 else if(p.endsWith('/publication'))data={published:false,profile:{name:'Three Studios',timezone:'UTC',services:[]}};
 else if(p.endsWith('/staff')||p.endsWith('/memberships'))data=[];
 await route.fulfill({status,contentType:'application/json',body:status===204?'':JSON.stringify(data)});
 });
 await page.goto('http://localhost:3000/business');
 const card=page.locator('.workspace-catalog-card').filter({has:page.getByText('Photography',{exact:true})});
 const garden=card.getByRole('button',{name:'Garden branch',exact:true});await garden.waitFor();assert.equal(await garden.getAttribute('aria-pressed'),'false');await garden.click();await page.waitForFunction(()=>[...document.querySelectorAll('.workspace-catalog-card button')].some(b=>b.textContent==='Garden branch'&&b.getAttribute('aria-pressed')==='true'));
 await page.getByRole('button',{name:/Hours & availability/}).click();
 const hours=page.getByRole('region',{name:'Opening hours',exact:true});await hours.locator('.schedule-day summary').first().click();await hours.getByRole('button',{name:'Monday',exact:true}).waitFor();
 await hours.getByRole('button',{name:'Tuesday',exact:true}).click();assert.match(await hours.locator('.schedule-day summary').first().innerText(),/Tuesday/);assert.equal(await hours.locator('input[name="day-0"]').inputValue(),'2');
 for(const theme of ['light','dark'])for(const width of [320,768,1280]){await page.setViewportSize({width,height:1000});await page.evaluate(t=>document.documentElement.dataset.theme=t,theme);await page.waitForTimeout(100);assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);await hours.screenshot({path:path.join(process.env.BROWSER_OUTPUT_DIR||os.tmpdir(),`workspace-${theme}-${width}.png`)});}
 const branch=page.locator('.workspace-catalog-card').filter({has:page.locator('p').filter({hasText:/^Garden branch$/})});await branch.getByRole('button',{name:'Delete branch'}).click();await branch.getByRole('button',{name:'Keep item'}).click();assert.equal(deletes,0);await branch.getByRole('button',{name:'Delete branch'}).click();await branch.getByRole('button',{name:'Confirm delete'}).click();await branch.waitFor({state:'detached'});assert.equal(deletes,1);
 const main=page.locator('.workspace-catalog-card').filter({has:page.locator('p').filter({hasText:/^Main branch$/})}).first();await main.getByRole('button',{name:'Delete branch'}).click();await main.getByRole('button',{name:'Confirm delete'}).click();await page.getByText('This item has upcoming bookings. Cancel or move those bookings before deleting it.',{exact:true}).waitFor();
 const add=page.locator('details').filter({has:page.locator('summary').filter({hasText:/^Add service$/})}).last();await add.locator('summary').click();await add.getByRole('textbox',{name:'Name',exact:true}).fill('Whole-day studio');await add.getByRole('radio',{name:'Whole day',exact:true}).check();assert.equal(await add.locator('input[name="duration"]').inputValue(),'1440');assert.equal(await add.getByRole('spinbutton',{name:'Duration in minutes'}).count(),0);await add.getByRole('button',{name:'Create service',exact:true}).click();await page.getByText('Whole-day studio created.',{exact:true}).waitFor();assert.equal(createdService.all_day,true);assert.equal(createdService.duration_minutes,1440);
 assert.deepEqual(errors,[]);console.log('PASS workspace buttons, weekday form values, inline keep/delete, conflict message, responsive light/dark. Mock API.');
} finally {await browser.close();}
