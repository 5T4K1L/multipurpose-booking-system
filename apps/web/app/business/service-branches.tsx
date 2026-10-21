"use client";
import {useEffect,useRef,useState} from "react";
import {api,CatalogItem,readable} from "../bookings/api";

export default function ServiceBranches({businessID,serviceID,writable,revision}:{businessID:string;serviceID:string;writable:boolean;revision:number}){
 const [branches,setBranches]=useState<CatalogItem[]>([]),[linked,setLinked]=useState<string[]>([]),[busy,setBusy]=useState(false),[loaded,setLoaded]=useState(false),[message,setMessage]=useState("");
 const pending=useRef(false);
 const root=`/api/v1/businesses/${businessID}/catalog`,path=`${root}/services/${serviceID}/locations`;
 useEffect(()=>{const c=new AbortController();
  async function all<T>(url:string){const values:T[]=[];let cursor="";do{const result=await api<T[]>(`${url}?limit=100${cursor?`&cursor=${encodeURIComponent(cursor)}`:""}`,{signal:c.signal});values.push(...result.data);cursor=result.cursor;}while(cursor);return values;}
  Promise.all([all<CatalogItem>(`${root}/locations`),all<string>(path)]).then(([items,ids])=>{if(!c.signal.aborted){setBranches(items);setLinked(ids);setLoaded(true);}}).catch(e=>{if(!c.signal.aborted)setMessage(readable(e));});return()=>c.abort();
 },[root,path,revision]);
 async function toggle(id:string,checked:boolean){if(pending.current)return;pending.current=true;setBusy(true);setMessage("");try{await api(`${path}/${id}`,{method:checked?"PUT":"DELETE"});setLinked(ids=>checked?[...ids,id]:ids.filter(value=>value!==id));setMessage("Branches saved. Publish your profile to update what customers see.");}catch(e){setMessage(readable(e));}finally{pending.current=false;setBusy(false);}}
 return <fieldset disabled={!writable||busy||!loaded} className="my-3 rounded-lg border p-3"><legend className="px-1 text-sm font-medium">Available at</legend><p className="mb-2 text-sm text-slate-600">Choose the branches where customers can book this service. Each change saves immediately.</p><div className="workspace-choices">{branches.map(branch=><button type="button" key={branch.id} className="workspace-choice" aria-pressed={linked.includes(branch.id)} onClick={()=>void toggle(branch.id,!linked.includes(branch.id))}>{branch.name}</button>)}</div>{loaded&&!branches.length&&<p className="text-sm">Add a branch above, then choose it here.</p>}{message&&<p role="status" className="mt-2 text-sm">{message}</p>}</fieldset>;
}
