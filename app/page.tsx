"use client";

import HomeSetup from "./components/HomeSetup";

export default function HomePage() {
  return (
    <main className="min-h-screen bg-paper text-ink">
      <HomeSetup onCreateRoom={() => {}} />
    </main>
  );
}
