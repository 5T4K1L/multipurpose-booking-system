import {createRequire} from 'node:module';
import assert from 'node:assert/strict';
import path from 'node:path';
import os from 'node:os';
const load=createRequire(import.meta.url);
const {chromium}=load(process.env.PLAYWRIGHT_MODULE||'playwright-core');
const browser=await chromium.launch({executablePath:process.env.CHROME_PATH||'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true});
try {
 const context=await browser.newContext();const page=await context.newPage();const errors=[];
 page.on('pageerror',e=>errors.push(e.message));let signedIn=false,registerCalls=0;
 await page.route('**/api/v1/**',async route=>{
  const p=new URL(route.request().url()).pathname;let status=200,data={};
  if(p.endsWith('/auth/providers'))data={providers:['google']};
  else if(p.endsWith('/auth/me')){status=signedIn?200:401;data={user:{id:1,username:'User',email:'owner@example.test'}};}
  else if(p.endsWith('/auth/login')){status=401;data={error:{code:'INVALID_CREDENTIALS'}};}
  else if(p.endsWith('/auth/register')){registerCalls++;status=409;data={error:{code:'EMAIL_ALREADY_REGISTERED'}};}
  else if(p==='/api/v1/businesses')data=[{id:'owned',name:'Wedding Studio',role:'BUSINESS_OWNER',timezone:'UTC'}];
  await route.fulfill({status,contentType:'application/json',body:JSON.stringify(data)});
 });
 await page.goto('http://localhost:3000/login');
 await page.getByRole('button',{name:'Continue with Google',exact:true}).waitFor();
 assert.equal(await page.getByRole('link',{name:'Sign out',exact:true}).count(),0);
 assert.equal(await page.getByRole('link',{name:'Your account',exact:true}).count(),0);
 await page.getByRole('button',{name:'Sign in',exact:true}).click();
 await page.getByText('Enter your email address.',{exact:true}).waitFor();
 await page.getByLabel('Email',{exact:true}).fill('owner@example.test');
 await page.getByLabel('Password',{exact:true}).fill('incorrect password');
 await page.getByRole('button',{name:'Sign in',exact:true}).click();
 await page.locator('#password-error').filter({hasText:'Email not registered or password incorrect'}).waitFor();
 assert.ok((await page.getByRole('button',{name:'Continue with Google'}).boundingBox()).y>(await page.getByRole('button',{name:'Sign in',exact:true}).boundingBox()).y);
 await page.goto('http://localhost:3000/register');
 await page.getByLabel('Name',{exact:true}).fill('Owner');
 await page.getByLabel('Email',{exact:true}).fill('invalid');
 await page.locator('#email-error').waitFor();
 await page.getByLabel('Email',{exact:true}).fill('owner@example.test');
 assert.equal(await page.locator('#email-error').count(),0);
 await page.getByLabel('Password',{exact:true}).fill('short');await page.locator('#password-error').waitFor();
 await page.getByLabel('Password',{exact:true}).fill('a long enough password');
 await page.getByLabel('Repeat password',{exact:true}).fill('different password');await page.getByText('Passwords do not match.',{exact:true}).waitFor();
 await page.getByRole('button',{name:'Create account',exact:true}).click();assert.equal(registerCalls,0);
 await page.getByLabel('Repeat password',{exact:true}).fill('a long enough password');assert.equal(await page.locator('#repeat-error').count(),0);
 await page.getByRole('checkbox').check();await page.getByRole('button',{name:'Create account',exact:true}).click();
 await page.locator('#email-error').filter({hasText:'already registered'}).waitFor();assert.equal(registerCalls,1);
 assert.ok((await page.getByRole('button',{name:'Continue with Google'}).boundingBox()).y>(await page.getByRole('button',{name:'Create account',exact:true}).boundingBox()).y);
 for(const route of ['login','register'])for(const width of [320,768,1280]){
  await page.setViewportSize({width,height:900});await page.goto('http://localhost:3000/'+route);
  await page.getByRole('button',{name:'Continue with Google'}).waitFor();
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  await page.screenshot({path:path.join(process.env.BROWSER_OUTPUT_DIR||os.tmpdir(),`owner-${route}-${width}.png`),fullPage:true});
 }
 signedIn=true;await page.goto('http://localhost:3000/account');
 await page.getByRole('heading',{name:'Wedding Studio',exact:true}).waitFor();
 assert.equal(await page.getByRole('heading',{name:'User',exact:true}).count(),0);
 assert.deepEqual(errors,[]);
 console.log('PASS inline/live validation, password confirmation, duplicate-account feedback, signed-out links, Google order, business account name and responsive forms. Mock API used.');
} finally {await browser.close();}
