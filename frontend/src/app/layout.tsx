// frontend/src/app/layout.tsx
import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import Header from "@/components/layout/Header";
import Footer from "@/components/layout/Footer";
import { Toaster } from "@/components/ui/sonner";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "arch-search | Find Arch Linux Solutions",
  description: "AI-powered search engine for Arch Linux troubleshooting. Find solutions to your errors instantly using content from the Arch Wiki.",
  keywords: "arch linux, troubleshooting, error search, arch wiki, linux support",
  authors: [{ name: "arch-search" }],
  openGraph: {
    title: "arch-search | Find Arch Linux Solutions",
    description: "AI-powered search engine for Arch Linux troubleshooting",
    type: "website",
    siteName: "arch-search",
  },
  twitter: {
    card: "summary_large_image",
    title: "arch-search | Find Arch Linux Solutions",
    description: "AI-powered search engine for Arch Linux troubleshooting",
  },
  viewport: "width=device-width, initial-scale=1",
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#ffffff" },
    { media: "(prefers-color-scheme: dark)", color: "#0a0a0a" },
  ],
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <link rel="icon" href="/favicon.ico" />
        <link rel="apple-touch-icon" href="/apple-touch-icon.png" />
        <meta name="theme-color" content="#1793d1" />
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased min-h-screen flex flex-col`}
      >
        <Header />
        <div className="flex-1">
          {children}
        </div>
        <Footer />
        <Toaster />
      </body>
    </html>
  );
}