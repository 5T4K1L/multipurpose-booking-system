"use client";

import {useNavigationSession} from "./navigation-session";
import Link from "next/link";
import {usePathname} from "next/navigation";
import {useEffect,useRef,useSyncExternalStore} from "react";

type Theme = "system" | "light" | "dark";
let fallback: Theme = "system";
function preference(): Theme {
  try { const value=localStorage.getItem("booking-appearance"); return value==="dark"||value==="light"?value:"system"; }
  catch {return fallback;}
}
function subscribe(callback:()=>void) {
  window.addEventListener("storage",callback);
  window.addEventListener("booking-appearance",callback);
  return ()=>{window.removeEventListener("storage",callback);window.removeEventListener("booking-appearance",callback);};
}
function applyTheme(value:Theme) {
  document.documentElement.dataset.theme=value==="system"?(matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light"):value;
}
function ThemeIcon({theme}:{theme:Theme}) {
  return <svg aria-hidden="true" viewBox="0 0 24 24" fill="none">
    {theme==="light"?<><circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19M5 19l1.5-1.5m11-11L19 5"/></>:theme==="dark"?<path d="M20.5 13A8.5 8.5 0 0 1 11 3.5 8.5 8.5 0 1 0 20.5 13Z"/>:<><rect x="3" y="4" width="18" height="13" rx="2"/><path d="M8 21h8m-4-4v4"/></>}
  </svg>;
}
function AppearanceControl({theme}:{theme:Theme}) {
  const panel=useRef<HTMLDivElement>(null);
  const trigger=useRef<HTMLButtonElement>(null);
  const choices=[{value:"light",label:"Light",description:"Bright and clear"},{value:"dark",label:"Dark",description:"Easy on the eyes"},{value:"system",label:"System",description:"Follow your device"}] as const;
  return <div className="appearance-switcher">
    <button ref={trigger} type="button" className="appearance-trigger" popoverTarget="appearance-options" aria-label="Appearance" title={`Appearance: ${theme}`}>
      <ThemeIcon theme={theme}/><span>{choices.find(choice=>choice.value===theme)?.label}</span><svg className="appearance-chevron" aria-hidden="true" viewBox="0 0 24 24" fill="none"><path d="m8 10 4 4 4-4"/></svg>
    </button>
    <div ref={panel} id="appearance-options" popover="auto" className="appearance-popover">
      <fieldset><legend>Appearance</legend><p className="appearance-caption">Choose how Booking looks on this device.</p>
        {choices.map(choice=><label key={choice.value} className="appearance-option" data-selected={theme===choice.value}>
          <ThemeIcon theme={choice.value}/><span><strong>{choice.label}</strong><small>{choice.description}</small></span>
          <input type="radio" name="booking-appearance" value={choice.value} checked={theme===choice.value} aria-label={choice.label} onChange={()=>{
            fallback=choice.value;try{localStorage.setItem("booking-appearance",choice.value);}catch{}
            applyTheme(choice.value);window.dispatchEvent(new Event("booking-appearance"));panel.current?.hidePopover();trigger.current?.focus();
          }}/>
        </label>)}
      </fieldset>
    </div>
  </div>;
}
export default function SiteHeader(){
  const path=usePathname();
  const session=useNavigationSession();
  const theme=useSyncExternalStore(subscribe,preference,()=>"system" as Theme);
  useEffect(()=>{
    const media=matchMedia("(prefers-color-scheme: dark)");
    const apply=()=>applyTheme(theme);apply();media.addEventListener("change",apply);
    return()=>media.removeEventListener("change",apply);
  },[theme]);
  const links=[{href:"/businesses",label:"Explore"},{href:"/my-bookings",label:"My bookings"},...(session.businesses.length?[{href:"/business",label:"Workspace"}]:[])];
  const active=(href:string)=>path===href||path.startsWith(href+"/");
  return <header className="site-header">
    <div className="site-header-inner">
      <Link href="/" className="site-brand" aria-label="Booking Platform home"><span className="brand-symbol" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none"><rect x="4" y="5" width="16" height="16" rx="5"/><path d="M8 3v5m8-5v5M8 14l3 3 5-6"/></svg></span><span>Booking<span className="brand-secondary"> Platform</span></span></Link>
      <nav className="desktop-nav" aria-label="Primary navigation">{links.map(link=><Link key={link.href} href={link.href} aria-current={active(link.href)?"page":undefined}>{link.label}</Link>)}</nav>
      <div className="header-actions"><AppearanceControl theme={theme}/>
      <Link href={session.signedIn?"/account":"/login"} className="header-account">{session.signedIn?"Account":"Business sign in"} <span aria-hidden="true">&#8599;</span></Link>
      <details className="mobile-menu" key={path}><summary aria-label="Open navigation menu"><svg aria-hidden="true" viewBox="0 0 24 24" fill="none"><path d="M4 7h16M4 12h16M4 17h16"/></svg></summary><nav aria-label="Mobile navigation">{links.map(link=><Link key={link.href} href={link.href} aria-current={active(link.href)?"page":undefined}>{link.label}</Link>)}<Link href={session.signedIn?"/account":"/login"}>{session.signedIn?"Account":"Business sign in"}</Link></nav></details>
      </div>
    </div>
  </header>;
}
