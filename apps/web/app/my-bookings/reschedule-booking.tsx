"use client";
import Link from "next/link";
import {useEffect,useRef,useState} from "react";
import {api,ApiError,mutate,readable,Slot} from "../bookings/api";
import {formatBookingTime} from "../bookings/format-booking-time";
type Result={booking:{id:string};replacement:{id:string;starts_at:string;ends_at:string};replayed:boolean};
export default function RescheduleBooking({id,disabled,canMove,onBusy,onMoved}:{id:string;disabled:boolean;canMove:boolean;onBusy:(value:boolean)=>void;onMoved:(replacement:string)=>void}){
 const [open,setOpen]=useState(false),[date,setDate]=useState(""),[zone,setZone]=useState(""),[slots,setSlots]=useState<Slot[]>([]),[selected,setSelected]=useState<Slot|null>(null),[message,setMessage]=useState(""),[loading,setLoading]=useState(false),[busy,setBusy]=useState(false),[uncertain,setUncertain]=useState(false),[version,setVersion]=useState(0),[receipt,setReceipt]=useState<Result|null>(null);
 const pending=useRef(false);
 useEffect(()=>{
  if(!open||!date||busy||uncertain||receipt)return;
  const c=new AbortController();let running=false;
  async function refresh(){if(running||document.hidden)return;running=true;setLoading(true);
   try{const r=await api<{timezone:string;slots:Slot[]}>(`/api/v1/customer/bookings/${id}/reschedule?date=${date}`,{signal:AbortSignal.any([c.signal,AbortSignal.timeout(15000)])});if(!c.signal.aborted){setSlots(r.data.slots);setZone(r.data.timezone);setMessage("");}}
   catch(e){if(!c.signal.aborted){setSlots([]);setMessage(readable(e));}}
   finally{running=false;if(!c.signal.aborted)setLoading(false);}
  }
  void refresh();const timer=window.setInterval(()=>void refresh(),30000);const focus=()=>void refresh();window.addEventListener("focus",focus);
  return()=>{c.abort();clearInterval(timer);window.removeEventListener("focus",focus);};
 },[id,open,date,busy,uncertain,receipt,version]);
 const available=!!selected&&slots.some(s=>s.starts_at===selected.starts_at&&s.ends_at===selected.ends_at);
 async function move(){if(pending.current||!selected||(!available&&!uncertain))return;pending.current=true;setBusy(true);onBusy(true);setMessage("");let submitted=false;
 try{const data={user:{id:0}};submitted=true;const result=await mutate<Result>(data.user.id,`/api/v1/customer/bookings/${id}/reschedule`,{starts_at:selected.starts_at});setReceipt(result);setUncertain(false);onMoved(result.replacement.id);}
 catch(e){const unknown=submitted&&(!(e instanceof ApiError)||e.status>=500||e.status===429);setUncertain(unknown);setMessage(unknown?"The result is uncertain. Retry this same move before choosing another time.":e instanceof ApiError&&e.status===409?"That time or booking is no longer available for this move. Refresh the times or booking details.":readable(e));if(!unknown)setVersion(v=>v+1);}
 finally{pending.current=false;setBusy(false);onBusy(false);}
 }
 if(receipt)return <section className="mt-5 rounded-lg border p-4" aria-label="Reschedule booking"><p role="status">Your booking has been rescheduled. The business can see the updated time.</p><p className="my-2">{formatBookingTime(receipt.replacement.starts_at,receipt.replacement.ends_at)} (your local time)</p><Link className="underline" href={`/my-bookings/${receipt.replacement.id}`}>View replacement booking</Link></section>;
 if(!canMove)return null;
 return <section className="mt-5 rounded-lg border p-4" aria-label="Reschedule booking">
 {!open?<button type="button" disabled={disabled} className="rounded-lg border px-4 py-2 disabled:opacity-50" onClick={()=>setOpen(true)}>Reschedule booking</button>:<>
 <h3 className="font-semibold">Choose a new time</h3><p className="my-3 text-sm">Your service, branch, duration and assignments stay the same. Your original booking stays reserved until the move succeeds.</p>
 <fieldset disabled={busy||disabled||uncertain} className="min-w-0"><label className="block">New date at the branch{zone?` (${zone})`:""}<input aria-label="New date at the branch" className="mt-2 block w-full min-w-0 rounded-lg border p-3" type="date" min="2000-01-01" max="2100-12-31" value={date} onChange={e=>{setDate(e.target.value);setSelected(null);setSlots([]);}}/></label>
 <p className="my-2 text-sm">Time choices below use your device local time. Occupied times are not selectable.</p>
 {date&&<button type="button" disabled={loading} className="my-2 rounded-lg border px-3 py-2" onClick={()=>setVersion(v=>v+1)}>Refresh times</button>}
 <div className="grid gap-2 sm:grid-cols-2">{slots.map(s=><button type="button" key={s.starts_at} aria-pressed={selected?.starts_at===s.starts_at} className={`rounded-lg border p-3 text-left ${selected?.starts_at===s.starts_at?"border-teal-800 bg-teal-50":""}`} onClick={()=>setSelected(s)}>{formatBookingTime(s.starts_at,s.ends_at)}</button>)}</div>
 {date&&!loading&&!slots.length&&<p className="my-3">No available times for this date. Try another date.</p>}
 </fieldset>
 {selected&&<div className="mt-4 rounded-lg bg-teal-50 p-3"><p>New time: {formatBookingTime(selected.starts_at,selected.ends_at)} (your local time)</p><button type="button" disabled={busy||disabled||(!available&&!uncertain)} className="mt-3 rounded-lg bg-teal-800 px-4 py-2 text-white disabled:opacity-50" onClick={()=>void move()}>{busy?"Moving booking...":uncertain?"Retry reschedule":"Confirm reschedule"}</button>{!available&&!uncertain&&<p>This time is no longer available. Choose another time.</p>}</div>}
 {!uncertain&&<button type="button" disabled={busy||disabled} className="mt-3 underline" onClick={()=>{setOpen(false);setSelected(null);setMessage("");}}>Keep current booking</button>}
 </>}{message&&<p role="status" className="mt-3 text-sm">{message}</p>}
 </section>;
}
