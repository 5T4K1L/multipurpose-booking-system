"use client";

import {useRef, useState, type ReactNode} from "react";
import {api, ApiError, mutate, readable} from "../bookings/api";

type Row = {id:string;status:string};
type Action<T> = {key:string;label:string;status:string;eligible:(row:T)=>boolean;path:(row:T)=>string};
export default function BookingOrganizer<T extends Row>({rows,busy,actions,onBusy,onUpdated,children,paginated=false,actor,onReview}: {
 rows:T[];busy:boolean;actions:Action<T>[];onBusy:(value:boolean)=>void;onUpdated:(id:string,status:string)=>void;
 children:(row:T,review:boolean)=>ReactNode;paginated?:boolean;actor?:number;onReview?:(ids:string[])=>Promise<void>;
}) {
 const [filter,setFilter]=useState("all"),[selected,setSelected]=useState<string[]>([]),[review,setReview]=useState(false);
 const [confirm,setConfirm]=useState<string|null>(null),[results,setResults]=useState<string[]>([]);
 const pending=useRef(false);
 const selectedRows=rows.filter(row=>selected.includes(row.id));
 const visible=rows.filter(row=>(filter==="all"||row.status===filter)&&(!review||selected.includes(row.id)));
 const statuses=["all","confirmed","completed","cancelled","rescheduled",...new Set(rows.map(r=>r.status).filter(s=>!["confirmed","completed","cancelled","rescheduled"].includes(s)))];
 const action=actions.find(a=>a.key===confirm);
 const eligible=action?selectedRows.filter(action.eligible):[];
 async function run(){
  if(!action||pending.current||!eligible.length)return;
  pending.current=true;onBusy(true);setResults([]);
  const output:string[]=[];
  try{
   const {data}=action.path(eligible[0]).startsWith("/api/v1/customer/bookings")?{data:{user:{id:0}}}:await api<{user:{id:number}}>("/api/v1/auth/me");
   if(actor!==undefined&&actor!==data.user.id)throw new Error("Account changed");
   for(const row of eligible){
    try{await mutate(data.user.id,action.path(row),{});onUpdated(row.id,action.status);setSelected(ids=>ids.filter(id=>id!==row.id));output.push(`${row.id}: ${action.status}.`);}
    catch(error){output.push(`${row.id}: ${readable(error)}`);if(!(error instanceof ApiError)||[401,403,429].includes(error.status)||error.status>=500){output.push("Stopped. Remaining selections were not processed. Refresh before retrying.");break;}}
    setResults([...output]);
   }
  }catch(error){output.push(readable(error));}
  finally{setResults(output);setConfirm(null);pending.current=false;onBusy(false);}
 }
 return <div className="booking-organizer">
  <div className="booking-categories" role="group" aria-label="Booking status categories">{statuses.map(status=><button key={status} disabled={busy} aria-pressed={filter===status} onClick={()=>{setFilter(status);setSelected([]);setReview(false);setConfirm(null);}}>{status==="all"?"All bookings":status.charAt(0).toUpperCase()+status.slice(1)} <span>{status==="all"?rows.length:rows.filter(row=>row.status===status).length}</span></button>)}</div>
  <p className="booking-hint">{paginated?"Categories and selection cover loaded bookings. Load more to include older records.":"Choose a category to keep current bookings separate from history."}</p>
  {!!actions.length&&!!rows.length&&<div className="booking-toolbar"><label><input type="checkbox" disabled={busy||!visible.length} checked={!!visible.length&&visible.every(r=>selected.includes(r.id))} onChange={e=>{setSelected(e.target.checked?visible.map(r=>r.id):[]);setConfirm(null);}}/> Select visible bookings</label><span>{selectedRows.length} selected</span>{!!selectedRows.length&&<><button disabled={busy} onClick={async()=>{if(!review&&onReview)await onReview(selectedRows.map(row=>row.id));setReview(!review);setConfirm(null);}}>{review?"Show category":"Review selected / edit individually"}</button>{actions.map(a=><button key={a.key} disabled={busy||!selectedRows.some(a.eligible)} onClick={()=>setConfirm(a.key)}>{a.label}</button>)}<button disabled={busy} onClick={()=>{setSelected([]);setReview(false);setConfirm(null);}}>Clear selection</button></>}</div>}
  {action&&<section className="booking-confirm" aria-label="Confirm bulk action"><h3>{action.label}?</h3><p>{eligible.length} eligible bookings will be updated individually. {selectedRows.length-eligible.length} ineligible selections will be skipped. History is retained. Each booking is checked again when saved.</p><div className="booking-actions"><button disabled={busy||!eligible.length} onClick={()=>void run()}>{busy?"Updating bookings...":`Confirm ${action.label.toLowerCase()}`}</button><button disabled={busy} onClick={()=>setConfirm(null)}>Go back</button></div></section>}
  {!!results.length&&<div role="status" className="bulk-results"><strong>Bulk action results</strong><ul>{results.map((result,i)=><li key={i}>{result}</li>)}</ul></div>}
  {!visible.length&&!busy&&<div className="booking-empty"><h3>{review?"No selected bookings remain":"No bookings in this category"}</h3><p>{paginated?"Load more bookings to check older records.":"Choose another category or explore businesses to plan your next booking."}</p></div>}
  <div className="booking-records">{visible.map(row=><div key={row.id} className="booking-record" data-booking-id={row.id}>{!!actions.length&&<label className="booking-selection"><input type="checkbox" disabled={busy} checked={selected.includes(row.id)} aria-label={`Select booking ${row.id}`} onChange={e=>{setSelected(ids=>e.target.checked?[...ids,row.id]:ids.filter(id=>id!==row.id));setConfirm(null);}}/> Select booking</label>}{children(row,review)}</div>)}</div>
 </div>;
}
