"use client";
import Link from "next/link";
import {usePathname} from "next/navigation";
import {useNavigationSession} from "../components/navigation-session";
export default function BusinessLayout({children}:{children:React.ReactNode}){
 const path=usePathname();const session=useNavigationSession();
 const canManage=session.businesses.some(b=>["BUSINESS_OWNER","BUSINESS_ADMIN"].includes(b.role));
 if(!session.ready)return <div className="workspace-loading" role="status"><div className="skeleton w-48"/><p>Opening your workspace...</p></div>;
 if(!session.businesses.length){if(path==="/business"&&session.signedIn)return children;return <main className="access-panel"><p className="eyebrow">Business workspace</p><h1>A space for your business.</h1><p>Manage bookings and settings when you own or join a business. Your personal bookings live in My bookings.</p><div className="hero-actions"><Link className="cta cta-primary" href="/my-bookings">My bookings</Link><Link className="cta cta-secondary" href={session.signedIn?"/business":"/login"}>{session.signedIn?"Set up a business":"Sign in"}</Link></div></main>;}
 const links=[{href:"/business/bookings",label:"Bookings",icon:"01"},{href:"/business/calendar",label:"Calendar",icon:"02"},...(canManage?[{href:"/business/customers",label:"Customers",icon:"03"}]:[]),{href:"/business",label:canManage?"Business settings":"Business details",icon:"04"}];
 return <div className="workspace-shell"><aside className="workspace-sidebar"><p className="sidebar-label">BUSINESS WORKSPACE</p><nav aria-label="Workspace navigation">{links.map(link=><Link key={link.href} href={link.href} aria-current={path===link.href?"page":undefined}><span aria-hidden="true">{link.icon}</span>{link.label}</Link>)}</nav><div className="sidebar-foot"><p>Switch to your personal space</p><Link href="/my-bookings">My bookings <span aria-hidden="true">&#8599;</span></Link></div></aside><div className="workspace-content">{children}</div></div>;
}
