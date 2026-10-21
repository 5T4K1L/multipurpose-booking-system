"use client";
import {useRef,useState} from "react";
import Image from "next/image";
import {api,readable} from "../bookings/api";
import {usePendingSave} from "./drafts";
export default function BusinessImage({id}:{id:string}){
 const [file,setFile]=useState<File|null>(null),[busy,setBusy]=useState(false),[message,setMessage]=useState(""),[version,setVersion]=useState(0);
 const form=useRef<HTMLFormElement>(null);const path=`/api/v1/businesses/${id}/image`;
 async function save(){if(!file)return;setBusy(true);try{await api(path,{method:"PUT",body:file,headers:{"Content-Type":file.type}});setFile(null);form.current?.reset();setVersion(v=>v+1);setMessage("Photo saved. Review and publish your profile to show it to customers.")}finally{setBusy(false)}}
 usePendingSave(path,"Business photo",!!file,busy,form,save);
 return <section className="ui-panel p-5 mb-5" aria-label="Business photo"><h2 className="text-xl font-semibold">Business photo</h2><p className="my-3 text-sm text-slate-600">Show customers your space or your work. Choose a PNG or JPEG up to 2 MB. Photos appear publicly after you review and publish.</p>
 <Image key={version} unoptimized src={`${process.env.NEXT_PUBLIC_API_BASE_URL??'http://localhost:8080'}${path}?v=${version}`} width={1000} height={500} alt="Saved business photo" className="my-4 max-h-72 w-full rounded-xl object-cover" onError={e=>{e.currentTarget.style.display='none'}}/>
 <form ref={form} onSubmit={e=>{e.preventDefault();void save().catch(e=>setMessage(readable(e)))}}><fieldset disabled={busy} className="flex flex-wrap items-center gap-3"><label>Choose business photo<input type="file" accept="image/png,image/jpeg" className="mt-2 block w-full text-sm" onChange={e=>{const selected=e.target.files?.[0]??null;if(selected&&(selected.size>2*1024*1024||!['image/png','image/jpeg'].includes(selected.type))){setMessage('Choose a PNG or JPEG up to 2 MB.');setFile(null);e.target.value='';return}setFile(selected);setMessage('')}}/></label><button className="cta cta-secondary" disabled={!file}>Save photo</button><button type="button" className="cta cta-secondary" onClick={async()=>{if(!window.confirm('Remove the photo from your next published profile?'))return;setBusy(true);try{await api(path,{method:'DELETE'});setFile(null);form.current?.reset();setVersion(v=>v+1);setMessage('Photo removed from draft. Publish to update the public profile.')}catch(e){setMessage(readable(e))}finally{setBusy(false)}}}>Remove photo</button></fieldset></form>{message&&<p role="status" className="mt-3 text-sm">{message}</p>}</section>
}
