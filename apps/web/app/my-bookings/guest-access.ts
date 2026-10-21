// Guest capabilities are independent of owner sessions. A recovery link grants
// access to this device's booking collection; keep it private.
export function guestAccess(create=false):string {
 const incoming=new URLSearchParams(window.location.hash.slice(1)).get("access");
 if(incoming&&/^[A-Za-z0-9_-]{43}$/.test(incoming)){
  localStorage.setItem("booking-guest-access",incoming);history.replaceState(null,"",window.location.pathname+window.location.search);
 }
 let token=localStorage.getItem("booking-guest-access")??"";
 if(!/^[A-Za-z0-9_-]{43}$/.test(token))token="";
 if(!token&&create){const bytes=crypto.getRandomValues(new Uint8Array(32));token=btoa(String.fromCharCode(...bytes)).replaceAll("+","-").replaceAll("/","_").replace(/=+$/,"");localStorage.setItem("booking-guest-access",token);}
 return token;
}
export function guestRecoveryLink(){const token=guestAccess();return token?`${window.location.origin}/my-bookings#access=${token}`:"";}
