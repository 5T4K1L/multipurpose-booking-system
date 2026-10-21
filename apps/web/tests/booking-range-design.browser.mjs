import {createRequire} from 'node:module';
import assert from 'node:assert/strict';
import path from 'node:path';
import os from 'node:os';
const {chromium}=createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE||'playwright-core');
const browser=await chromium.launch({executablePath:'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true});
try{
 const context=await browser.newContext({timezoneId:'UTC'}),page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
 const date=new Date(Date.now()+86400000).toISOString().slice(0,10),next=new Date(Date.now()+2*86400000).toISOString().slice(0,10);
 const slots=[10,11,12].map(hour=>({starts_at:`${date}T${hour}:00:00Z`,ends_at:`${date}T${hour+1}:00:00Z`}));let checks=0;
 await page.route('**/api/v1/**',async route=>{const u=new URL(route.request().url());let data={},status=200;
 if(u.pathname.endsWith('/auth/me'))status=401;
 else if(u.pathname.endsWith('/calendar'))data={timezone:'UTC',month:date.slice(0,7),days:[{date,status:'available',slots}]};
 else if(u.pathname.endsWith('/range')){checks++;data={slots:[{starts_at:u.searchParams.get('starts_at'),ends_at:u.searchParams.get('ends_at')}]};}
 else if(u.pathname==='/api/v1/public/businesses/biz')data={id:'biz',name:'Studio',timezone:'UTC',services:[{id:'svc',name:'Photography',duration_minutes:60,booking_options:{enabled:true,minimum_minutes:60,maximum_minutes:2880},locations:[{id:'loc',name:'Main branch',timezone:'UTC'}]}]};
 await route.fulfill({status,contentType:'application/json',body:JSON.stringify(data)});});
 await page.goto('http://localhost:3000/businesses/biz');await page.getByLabel('Service',{exact:true}).selectOption('svc');
 await page.getByRole('button',{name:new RegExp('^'+date+':')}).click();
 const range=page.getByRole('region',{name:'Choose booking range'});await range.getByRole('heading',{name:'Choose your start and end'}).waitFor();
 assert.equal(await range.locator('select').count(),0);
 await range.getByRole('group',{name:'Quick booking lengths'}).getByRole('button',{name:'2 hr',exact:true}).click();assert.equal(await range.getByLabel('End time',{exact:true}).inputValue(),'12:00');
 await range.getByRole('button',{name:'Check availability',exact:true}).click();await range.getByText('The full date and time range is available. Confirm below to book.',{exact:true}).waitFor();assert.equal(checks,1);
 await range.getByRole('group',{name:'Available start times'}).getByRole('button').nth(1).click();assert.equal(await range.getByRole('button',{name:'Confirm booking',exact:true}).count(),0);
 await range.getByLabel('End date',{exact:true}).fill(next);await range.getByLabel('End time',{exact:true}).fill('11:00');
 await range.getByRole('button',{name:'Check availability',exact:true}).click();await range.getByText('The full date and time range is available. Confirm below to book.',{exact:true}).waitFor();assert.equal(checks,2);
 for(const theme of ['light','dark'])for(const width of [320,768,1280]){await page.setViewportSize({width,height:1000});await page.evaluate(theme=>document.documentElement.dataset.theme=theme,theme);await page.waitForTimeout(250);assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);await range.screenshot({path:path.join(process.env.BROWSER_OUTPUT_DIR||os.tmpdir(),`range-${theme}-${width}.png`)});}
 assert.deepEqual(errors,[]);console.log('PASS no dropdowns, start buttons, quick lengths, custom multi-day end, availability check, stale confirmation clearing, responsive light/dark layouts. Mock API used.');
}finally{await browser.close();}
