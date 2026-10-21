"use client";
import {createContext,useContext,useEffect,useState,type ReactNode} from "react";
import {usePathname} from "next/navigation";
import {api,Business} from "../bookings/api";
type Session={ready:boolean;signedIn:boolean;businesses:Business[]};
const Context=createContext<Session>({ready:false,signedIn:false,businesses:[]});
export function NavigationSession({children}:{children:ReactNode}){
 const path=usePathname();const [value,setValue]=useState<Session>({ready:false,signedIn:false,businesses:[]});
 useEffect(()=>{const controller=new AbortController();let generation=0;
 async function refresh(){const revision=++generation;let signedIn=false;
  try{await api("/api/v1/auth/me",{signal:controller.signal});signedIn=true;const businesses:Business[]=[];let cursor="";
   do{const result=await api<Business[]>(`/api/v1/businesses?limit=100${cursor?`&cursor=${encodeURIComponent(cursor)}`:""}`,{signal:controller.signal});businesses.push(...result.data);cursor=result.cursor;}while(cursor);
   if(!controller.signal.aborted&&revision===generation)setValue({ready:true,signedIn:true,businesses});
  }catch{if(!controller.signal.aborted&&revision===generation)setValue({ready:true,signedIn,businesses:[]});}
 }
 void refresh();window.addEventListener("focus",refresh);window.addEventListener("booking-session-changed",refresh);
 return()=>{controller.abort();window.removeEventListener("focus",refresh);window.removeEventListener("booking-session-changed",refresh);};
 },[path]);
 return <Context.Provider value={value}>{children}</Context.Provider>;
}
export const useNavigationSession=()=>useContext(Context);
