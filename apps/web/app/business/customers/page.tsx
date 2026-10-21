"use client";
import {useEffect,useState} from "react";
import {api,Business,readable} from "../../bookings/api";
import {button,input,panel} from "../styles";
type Customer={id:number;name:string;email:string;booking_records:number;confirmed_bookings:number;latest_booking:string};
export default function CustomersPage(){
 const [businesses,setBusinesses]=useState<Business[]>([]),[selected,setSelected]=useState(""),[message,setMessage]=useState("Loading businesses...");
 const [version,setVersion]=useState(0);
 useEffect(()=>{
  const c=new AbortController();
  async function load(){
   try{let cursor="";const all:Business[]=[];
    do{const r=await api<Business[]>(`/api/v1/businesses?limit=100${cursor?`&cursor=${encodeURIComponent(cursor)}`:""}`,{signal:c.signal});all.push(...r.data);cursor=r.cursor;}while(cursor);
    if(c.signal.aborted)return;const allowed=all.filter(b=>["BUSINESS_OWNER","BUSINESS_ADMIN"].includes(b.role));setBusinesses(allowed);setSelected(old=>allowed.some(b=>b.id===old)?old:(allowed.find(b=>b.role==="BUSINESS_OWNER")??allowed[0])?.id??"");setMessage(allowed.length?"":"Customer records are available to business owners and admins.");
   }catch(e){if(!c.signal.aborted){setBusinesses([]);setSelected("");setMessage(readable(e));}}
  }
  void load();return()=>c.abort();
 },[version]);
 useEffect(()=>{const refresh=()=>{setBusinesses([]);setSelected("");setVersion(v=>v+1);};window.addEventListener("focus",refresh);return()=>window.removeEventListener("focus",refresh);},[]);
 return <main className="min-h-screen bg-slate-50 px-4 py-8 text-slate-900 sm:px-8"><div className="mx-auto max-w-6xl">
 
 <h1 className="mt-7 text-3xl font-semibold">Customers</h1><p className="my-4">People who have booked with your business. Search their contact details and review their booking activity.</p>
 {message&&<p role="status" className="my-4">{message}</p>}
 {selected&&<p className="text-sm text-slate-600">Your business: <strong>{businesses.find(b=>b.id===selected)?.name}</strong></p>}
 {selected&&<CustomerTable key={selected} businessID={selected}/>}
 </div></main>;
}
function CustomerTable({businessID}:{businessID:string}){
 const [draft,setDraft]=useState(""),[search,setSearch]=useState(""),[rows,setRows]=useState<Customer[]>([]),[cursor,setCursor]=useState(""),[message,setMessage]=useState(""),[busy,setBusy]=useState(true),[version,setVersion]=useState(0);
 useEffect(()=>{const c=new AbortController();
 api<Customer[]>(`/api/v1/businesses/${businessID}/customers?limit=25&q=${encodeURIComponent(search)}`,{signal:c.signal})
 .then(r=>{if(!c.signal.aborted){setRows(r.data);setCursor(r.cursor);setMessage(r.data.length?"":"No customers found. Try another search, or check back after your first customer booking.");}})
 .catch(e=>{if(!c.signal.aborted)setMessage(readable(e));}).finally(()=>{if(!c.signal.aborted)setBusy(false);});return()=>c.abort();},[businessID,search,version]);
 function refresh(){setBusy(true);setRows([]);setCursor("");setMessage("");setVersion(v=>v+1);}
 async function more(){if(busy||!cursor)return;setBusy(true);try{const r=await api<Customer[]>(`/api/v1/businesses/${businessID}/customers?limit=25&q=${encodeURIComponent(search)}&cursor=${encodeURIComponent(cursor)}`);setRows(old=>[...old,...r.data.filter(row=>!old.some(v=>v.id===row.id))]);setCursor(r.cursor);setMessage("");}catch(e){setRows([]);setCursor("");setMessage(readable(e));}finally{setBusy(false);}}
 return <section className={`${panel} mt-6`} aria-label="Business customers">
 <form className="flex flex-wrap items-end gap-3" onSubmit={e=>{e.preventDefault();setSearch(draft.trim());refresh();}}><label className="w-full min-w-0 sm:w-auto sm:flex-1">Search name or email<input className={input} value={draft} maxLength={200} onChange={e=>setDraft(e.target.value)}/></label><button disabled={busy} className={button}>Search customers</button><button type="button" disabled={busy} className="workspace-text-action" onClick={refresh}>Refresh customers</button></form>
 <p className="my-3 text-sm">Booking records include cancelled and rescheduled entries. Counts only include this business. Times use your device local time.</p>
 {busy&&<p role="status">Loading customers...</p>}{message&&<p role="status" className="my-3">{message}</p>}
 <div className="customer-table-wrap" tabIndex={0} role="region" aria-label="Customer table"><table className="customer-table"><caption className="sr-only">Customers of this business</caption><thead><tr>{["Name","Email","Booking records","Confirmed","Latest booking time"].map(t=><th key={t} scope="col" className="border-b p-3">{t}</th>)}</tr></thead><tbody>{rows.map(row=><tr key={row.id}><td data-label="Name" className="border-b p-3">{row.name}</td><td data-label="Email" className="break-all border-b p-3">{row.email}</td><td data-label="Bookings" className="border-b p-3">{row.booking_records}</td><td data-label="Confirmed" className="border-b p-3">{row.confirmed_bookings}</td><td data-label="Latest booking" className="border-b p-3">{new Date(row.latest_booking).toLocaleString(undefined,{dateStyle:"medium",timeStyle:"short"})}</td></tr>)}</tbody></table></div>
 {cursor&&<button type="button" disabled={busy} className={`${button} mt-4`} onClick={()=>void more()}>Load more customers</button>}
 </section>;
}
