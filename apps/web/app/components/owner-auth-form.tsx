"use client";

import Link from "next/link";
import {useRouter} from "next/navigation";
import {useRef, useState, type FormEvent} from "react";
import OAuthButtons from "../login/oauth-buttons";
import {useNavigationSession} from "./navigation-session";

type Field = "username" | "email" | "password" | "repeat" | "owner";
export default function OwnerAuthForm({register=false}:{register?:boolean}) {
  const router=useRouter(), session=useNavigationSession();
  const locked=useRef(false);
  const [values,setValues]=useState({username:"",email:"",password:"",repeat:"",owner:false});
  const [touched,setTouched]=useState<Partial<Record<Field,boolean>>>({});
  const [submitted,setSubmitted]=useState(false),[pending,setPending]=useState(false),[show,setShow]=useState(false);
  const [message,setMessage]=useState(""),[failed,setFailed]=useState(false);
  const [serverErrors,setServerErrors]=useState<Partial<Record<Field,string>>>({});
  const errors:Partial<Record<Field,string>>={};
  if(register&&!values.username.trim())errors.username="Enter your name.";
  if(!values.email.trim())errors.email="Enter your email address.";
  else if(!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email.trim()))errors.email="Enter a valid email address, such as name@example.com.";
  if(!values.password)errors.password="Enter your password.";
  else if(register&&(Array.from(values.password).length<15||Array.from(values.password).length>128))errors.password="Use 15-128 characters for your password.";
  if(register&&values.repeat!==values.password)errors.repeat="Passwords do not match.";
  else if(register&&!values.repeat)errors.repeat="Repeat your password.";
  if(register&&!values.owner)errors.owner="Confirm that you are creating a business owner account.";
  function error(field:Field){return serverErrors[field]||((submitted||touched[field])?errors[field]:undefined);}
  function change(field:Exclude<Field,"owner">,value:string){
    setValues(old=>({...old,[field]:value}));setTouched(old=>({...old,[field]:true}));
    setServerErrors({});setMessage("");
  }
  async function submit(event:FormEvent<HTMLFormElement>){
    event.preventDefault();if(locked.current)return;setSubmitted(true);setMessage("");setServerErrors({});
    if(Object.keys(errors).length)return;
    locked.current=true;setPending(true);setFailed(false);
    try{
      const response=await fetch(`${process.env.NEXT_PUBLIC_API_BASE_URL??"http://localhost:8080"}/api/v1/auth/${register?"register":"login"}`,{
        method:"POST",headers:{"Content-Type":"application/json"},credentials:register?"omit":"include",signal:AbortSignal.timeout(15000),
        body:JSON.stringify({email:values.email.trim(),password:values.password,...(register?{username:values.username.trim()}:{})})
      });
      if(response.ok){
        if(register){setValues({username:"",email:"",password:"",repeat:"",owner:false});setTouched({});setSubmitted(false);setMessage("Account created. Sign in to set up your business.");}
        else{window.dispatchEvent(new Event("booking-session-changed"));router.push("/business");setMessage("You're signed in.");}
      }else{
        setFailed(true);
        if(register&&response.status===409)setServerErrors({email:"This email is already registered. Sign in, or continue with Google if you used Google."});
        else if(!register&&response.status===401)setServerErrors({password:"Email not registered or password incorrect. Check both, or continue with Google if you used Google."});
        else setMessage(response.status===429?"Too many attempts. Please wait before trying again.":response.status===400?"Check the details above and try again.":"This service is unavailable right now. Please try again later.");
      }
    }catch{setFailed(true);setMessage("We couldn't reach the server. Please try again.");}
    finally{locked.current=false;setPending(false);}
  }
  function field(name:Exclude<Field,"owner">,label:string,type:string,autoComplete:string){
    const problem=error(name);
    return <div key={name}><label htmlFor={name}>{label}</label><input id={name} name={name} type={type} autoComplete={autoComplete} value={values[name]} onChange={e=>change(name,e.target.value)} onBlur={()=>setTouched(old=>({...old,[name]:true}))} required maxLength={name==="email"?254:name==="username"?80:512} aria-invalid={!!problem} aria-describedby={[problem?`${name}-error`:"",register&&name==="password"?"password-help":""].filter(Boolean).join(" ")||undefined} className="ui-input mt-2 w-full border px-3 py-3"/>{register&&name==="password"&&<p id="password-help" className="mt-2 text-sm">15-128 characters. Spaces are welcome.</p>}{problem&&<p id={`${name}-error`} role="alert" className="mt-2 text-sm text-red-700">{problem}</p>}</div>;
  }
  return <main className="auth-page min-h-screen"><section className="auth-card" aria-labelledby="auth-title">
    <Link href="/" className="text-sm underline underline-offset-4">Booking Platform</Link>
    <h1 id="auth-title" className="mt-6 text-3xl font-semibold">{register?"Create your business owner account":"Business owner sign in"}</h1>
    <p className="mt-3 text-sm">{register?"Create an owner account, then set up your business profile. Customers can book without an account.":"Manage your business here. Customers can book without an account."}</p>
    <form noValidate onSubmit={submit} className="mt-8 space-y-5" aria-busy={pending}>
      <fieldset disabled={pending} className="space-y-5">
        {register&&field("username","Name","text","name")}
        {field("email","Email","email",register?"email":"username")}
        {field("password","Password",show?"text":"password",register?"new-password":"current-password")}
        {register&&field("repeat","Repeat password",show?"text":"password","new-password")}
        <button type="button" className="min-h-11 text-sm underline" aria-controls={register?"password repeat":"password"} aria-pressed={show} onClick={()=>setShow(!show)}>{show?"Hide password":"Show password"}</button>
        {register&&<div><label className="flex items-start gap-3 text-sm"><input type="checkbox" checked={values.owner} onChange={e=>{setValues(old=>({...old,owner:e.target.checked}));setTouched(old=>({...old,owner:true}));}} aria-invalid={!!error("owner")} aria-describedby={error("owner")?"owner-error":undefined} className="mt-1"/>I am creating an account to own and manage a business.</label>{error("owner")&&<p id="owner-error" role="alert" className="mt-2 text-sm text-red-700">{error("owner")}</p>}</div>}
        <button className="cta cta-primary min-h-12 w-full" type="submit">{pending?(register?"Creating account...":"Signing in..."):(register?"Create account":"Sign in")}</button>
      </fieldset>
      {message&&<p role={failed?"alert":"status"} className="text-sm">{message}</p>}
    </form>
    <OAuthButtons disabled={pending}/>
    <p className="mt-6 text-sm">{register?"Already have an account? ":"New here? "}<Link href={register?"/login":"/register"} className="underline underline-offset-4">{register?"Sign in":"Create an account"}</Link></p>
    {session.signedIn&&<div className="mt-4 flex gap-4 text-sm"><Link href="/account" className="underline">Your account</Link><Link href="/logout" className="underline">Sign out</Link></div>}
  </section></main>;
}
