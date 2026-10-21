import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import {NavigationSession} from "./components/navigation-session";
import SiteHeader from "./components/site-header";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: {default: "Booking Platform", template: "%s | Booking Platform"},
  description: "Find a business, choose a time, and keep your bookings organized.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <head><script dangerouslySetInnerHTML={{__html: `(function(){try{var t=localStorage.getItem('booking-appearance');document.documentElement.dataset.theme=t==='dark'||t==='light'?t:matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light'}catch(e){document.documentElement.dataset.theme=matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light'}})()`}} /></head>
      <body className="min-h-full flex flex-col">
        <a href="#main-content" className="skip-link">Skip to content</a>
        <NavigationSession>
        <SiteHeader />
        <div id="main-content" tabIndex={-1} className="app-content">{children}</div>
        </NavigationSession>
        <footer className="site-footer"><span>Booking Platform</span><span>A little less planning. A little more living.</span></footer>
      </body>
    </html>
  );
}
