"use client";
import {useState} from "react";
import {input} from "./styles";
export default function ServiceDuration({allDay=false,minutes=30}:{allDay?:boolean;minutes?:number}){
 const [whole,setWhole]=useState(allDay);
 return <fieldset className="service-duration"><legend className="font-medium">Service length</legend><div className="duration-modes">
 {[{value:false,label:"Timed appointment"},{value:true,label:"Whole day"}].map(mode=><label key={mode.label} className="workspace-choice workspace-mode"><input type="radio" name="duration_mode" value={mode.value?"whole_day":"timed"} checked={whole===mode.value} onChange={()=>setWhole(mode.value)}/>{mode.label}</label>)}
 </div><input type="hidden" name="all_day" value={String(whole)}/>{whole?<><input type="hidden" name="duration" value="1440"/><p className="mt-3 text-sm text-slate-600">From opening to closing on the selected date, using the branch&apos;s local hours.</p></>:<label className="mt-3 block">Duration in minutes<input name="duration" type="number" defaultValue={allDay?30:minutes} required min={1} max={10080} step={1} className={input}/></label>}</fieldset>;
}
