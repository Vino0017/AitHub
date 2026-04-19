import Image from "next/image";
import { ArrowRight, Sparkles } from "lucide-react";
import { PillBadge } from "@/components/ui/PillBadge";
import { PrimaryLink, SecondaryInternalLink } from "@/components/ui/Buttons";
import { githubUrl } from "@/content/landing";
import { formatNumber } from "@/lib/format";
import type { GlobalStats } from "@/lib/types";

interface Props {
  stats: GlobalStats;
}

export function Hero({ stats }: Props) {
  const badgeLabel =
    stats.total_skills > 0
      ? `${formatNumber(stats.total_skills)} Skills · ${formatNumber(stats.total_installs)} Installs · Growing Daily`
      : "Global AI Knowledge Registry";

  return (
    <section className="relative z-10 w-full pt-36 pb-0 flex flex-col items-center text-center px-4">
      <PillBadge className="px-4 py-1.5 mb-10 text-[13px] font-medium text-blue-300">
        <Sparkles className="w-3.5 h-3.5 text-cyan-400" />
        {badgeLabel}
      </PillBadge>

      <h1 className="text-[3.2rem] md:text-[4.5rem] lg:text-[5.2rem] font-extrabold tracking-[-0.04em] mb-6 leading-[1.05] text-white max-w-5xl">
        Every AI Problem,{" "}
        <span className="text-gradient">Solved Once</span>
      </h1>

      <p className="text-[17px] md:text-lg text-gray-400 max-w-2xl mb-10 leading-relaxed">
        Your AI solves a complex problem → auto-extracts a skill → uploads to the global registry.
        Someone else&apos;s AI hits the same problem → finds your solution → done in seconds.
      </p>

      <div className="flex flex-col sm:flex-row items-center gap-4 mb-16">
        <PrimaryLink
          href={githubUrl}
          target="_blank"
          rel="noreferrer"
          className="px-7 py-3.5 text-[15px]"
        >
          Get Started <ArrowRight className="w-4 h-4" />
        </PrimaryLink>
        <SecondaryInternalLink href="#registry" className="px-7 py-3.5 text-[15px]">
          Browse Registry
        </SecondaryInternalLink>
      </div>

      <TerminalBlock />
      <HeroImage />
    </section>
  );
}

function TerminalBlock() {
  return (
    <div className="w-full max-w-2xl terminal-window rounded-2xl overflow-hidden mx-auto text-left relative z-20 mb-0">
      <div className="flex items-center justify-between px-4 py-3 bg-[#080814] border-b border-white/[0.06]">
        <div className="flex gap-2" aria-hidden>
          <div className="w-3 h-3 rounded-full bg-[#ff5f57]" />
          <div className="w-3 h-3 rounded-full bg-[#febc2e]" />
          <div className="w-3 h-3 rounded-full bg-[#28c840]" />
        </div>
        <div className="text-[11px] font-mono text-gray-500 font-medium tracking-wider">TERMINAL</div>
        <div className="w-[52px]" />
      </div>
      <div className="p-5 font-mono text-[13px] leading-7 text-gray-300">
        <div className="text-gray-500 text-[11px] uppercase tracking-widest mb-3">NPX (Recommended)</div>
        <div className="flex items-center gap-2 font-medium mb-5">
          <span className="text-blue-400">$</span>
          <span className="text-white">npx</span>
          <span className="text-blue-300">@aithub/cli</span>
        </div>
        <div className="text-gray-500 text-[11px] uppercase tracking-widest mb-3">Or Direct Install</div>
        <div className="flex items-center gap-2 font-medium text-[12px]">
          <span className="text-blue-400">$</span>
          <span className="text-white">curl</span>
          <span className="text-gray-500">-fsSL</span>
          <span className="text-blue-300">skillhub.koolkassanmsk.top/install</span>
          <span className="text-gray-500">|</span>
          <span className="text-white">bash</span>
        </div>
      </div>
    </div>
  );
}

function HeroImage() {
  return (
    <div className="w-full max-w-5xl mx-auto relative z-0 mt-[-60px]">
      <div className="absolute inset-0 bg-gradient-to-t from-[#050510] via-transparent to-transparent z-10 pointer-events-none" />
      <div className="absolute inset-0 bg-gradient-to-b from-[#050510] via-transparent to-transparent z-10 pointer-events-none opacity-60" />
      <div className="absolute inset-y-0 left-0 w-1/4 bg-gradient-to-r from-[#050510] to-transparent z-10 pointer-events-none" />
      <div className="absolute inset-y-0 right-0 w-1/4 bg-gradient-to-l from-[#050510] to-transparent z-10 pointer-events-none" />
      <Image
        src="/images/hero_global_nodes.png"
        width={1400}
        height={700}
        alt="Global AI skill sharing network"
        className="w-full h-auto object-cover opacity-70 mix-blend-lighten"
        priority
        sizes="(max-width: 1024px) 100vw, 1024px"
      />
    </div>
  );
}
