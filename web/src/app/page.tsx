import { Nav } from "@/components/landing/Nav";
import { Hero } from "@/components/landing/Hero";
import { StatsBar } from "@/components/landing/StatsBar";
import { HowItWorks } from "@/components/landing/HowItWorks";
import { Registry } from "@/components/landing/Registry";
import { CtaFooter } from "@/components/landing/CtaFooter";
import { getGlobalStats, getSkills } from "@/lib/api";

export const revalidate = 60;

export default async function Home() {
  const [skillData, stats] = await Promise.all([getSkills(), getGlobalStats()]);

  return (
    <main id="main" className="relative min-h-screen bg-[var(--paper)]">
      <div className="relative z-2">
        <Nav />
        <Hero stats={stats} />
        <StatsBar stats={stats} />
        <HowItWorks />
        <Registry skills={skillData.skills} totalSkills={skillData.total} />
        <CtaFooter />
      </div>
    </main>
  );
}
